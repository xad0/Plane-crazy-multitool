package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
)

type clearCopyToastMsg struct {
	id int
}

type copyHitTarget struct {
	x1       int
	x2       int
	y        int
	rawValue string
}

type copyState struct {
	hoverX       int
	hoverY       int
	hasHover     bool
	toastText    string
	toastSuccess bool
	toastTimerID int
	targets      []copyHitTarget
	markedValues map[int]string
	nextMarkID   int
}

func (c *copyState) clearTargets() {
	c.targets = c.targets[:0]
	c.markedValues = make(map[int]string)
	c.nextMarkID = 0
}

// copyMarkerToken uses only control bytes, so its visible terminal width is
// zero. Markers survive Lip Gloss layout and are stripped before the view is sent.
const copyMarkerPrefix = "\x1b]777;pc-multitool-copy;"

// OSC markers have zero terminal width and do not inject layout-changing C0
// bytes. They are stripped after the full view is composed, when final screen
// coordinates are known. Do not replace these with raw control-byte IDs:
// Lipgloss processes those bytes as ordinary content and panel widths drift.
func copyMarkerToken(id int, opening bool) string {
	kind := "E"
	if opening {
		kind = "S"
	}
	return fmt.Sprintf("%s%s;%05d\x1b\\", copyMarkerPrefix, kind, id)
}

func findCopyMarker(content string, from int) (start, end, id int, opening, ok bool) {
	if from < 0 {
		from = 0
	}
	if from >= len(content) {
		return 0, 0, 0, false, false
	}
	relative := strings.Index(content[from:], copyMarkerPrefix)
	if relative < 0 {
		return 0, 0, 0, false, false
	}
	start = from + relative
	payloadStart := start + len(copyMarkerPrefix)
	terminator := strings.Index(content[payloadStart:], "\x1b\\")
	if terminator < 0 {
		return 0, 0, 0, false, false
	}
	payloadEnd := payloadStart + terminator
	parts := strings.Split(content[payloadStart:payloadEnd], ";")
	if len(parts) != 2 || (parts[0] != "S" && parts[0] != "E") {
		return 0, 0, 0, false, false
	}
	parsedID, err := strconv.Atoi(parts[1])
	if err != nil || parsedID < 1 || parsedID > 0xffff {
		return 0, 0, 0, false, false
	}
	return start, payloadEnd + 2, parsedID, parts[0] == "S", true
}

func (c *copyState) markValue(rawValue, displayText string) string {
	if rawValue == "" || displayText == "" {
		return displayText
	}
	c.nextMarkID++
	if c.nextMarkID > 0xffff {
		c.nextMarkID = 1
	}
	id := c.nextMarkID
	if c.markedValues == nil {
		c.markedValues = make(map[int]string)
	}
	c.markedValues[id] = rawValue
	return copyMarkerToken(id, true) + displayText + copyMarkerToken(id, false)
}

// finalizeContent converts markers into hit targets using the final composed
// screen coordinates. It also adds hover highlighting without panel-specific offsets.
// copyActiveSGRState returns the SGR style state active at the end of content.
// A hover span's Lipgloss Render ends with SGR reset; re-applying this state
// prevents that reset from erasing the panel background for the rest of a row.
func copyActiveSGRState(content string) string {
	var state strings.Builder

	for i := 0; i < len(content); {
		if content[i] != '\x1b' || i+1 >= len(content) {
			i++
			continue
		}

		switch content[i+1] {
		case '[':
			j := i + 2
			for j < len(content) && (content[j] < 0x40 || content[j] > 0x7e) {
				j++
			}
			if j >= len(content) {
				i = len(content)
				continue
			}

			if content[j] == 'm' {
				sequence := content[i : j+1]
				params := content[i+2 : j]
				resets := params == ""
				for _, param := range strings.Split(params, ";") {
					if param == "" || param == "0" {
						resets = true
						break
					}
				}
				if resets {
					state.Reset()
				}
				state.WriteString(sequence)
			}
			i = j + 1

		case ']':
			// Skip OSC strings, terminated by BEL or ST (ESC \\).
			j := i + 2
			for j < len(content) {
				if content[j] == '\a' {
					j++
					break
				}
				if content[j] == '\x1b' && j+1 < len(content) && content[j+1] == '\\' {
					j += 2
					break
				}
				j++
			}
			i = j

		default:
			i += 2
		}
	}

	return state.String()
}

func (c *copyState) finalizeContent(content string, p palette) string {
	var output strings.Builder
	output.Grow(len(content))
	cursor := 0

	for cursor < len(content) {
		start, startEnd, id, opening, found := findCopyMarker(content, cursor)
		if !found {
			output.WriteString(content[cursor:])
			break
		}
		if !opening {
			output.WriteString(content[cursor:start])
			cursor = startEnd
			continue
		}

		closeStart, closeEnd := -1, -1
		search := startEnd
		for search < len(content) {
			pos, end, closeID, isOpen, ok := findCopyMarker(content, search)
			if !ok {
				break
			}
			if !isOpen && closeID == id {
				closeStart, closeEnd = pos, end
				break
			}
			search = end
		}
		if closeStart < 0 {
			// Malformed marker: drop its start token and preserve remaining text.
			output.WriteString(content[cursor:start])
			cursor = startEnd
			continue
		}

		output.WriteString(content[cursor:start])
		prefix := output.String()
		y := strings.Count(prefix, "\n")
		lineStart := strings.LastIndex(prefix, "\n") + 1
		x := lipgloss.Width(prefix[lineStart:])
		display := content[startEnd:closeStart]
		width := lipgloss.Width(display)

		if raw, exists := c.markedValues[id]; exists && width > 0 {
			c.addTarget(x, y, width, raw)
			if c.isHovered(x, y, width) {
				// Preserve the SGR state active before this value. Lipgloss's
				// hover style resets SGR at its end; without restoration, the
				// panel background disappears for the remainder of the row.
				activeStyle := copyActiveSGRState(content[:start])
				hovered := lipgloss.NewStyle().
					Background(p.accent).
					Foreground(p.background).
					Bold(true).
					Underline(true).
					Render(display)
				output.WriteString(hovered)
				output.WriteString(activeStyle)
			} else {
				output.WriteString(display)
			}
		} else {
			output.WriteString(display)
		}
		cursor = closeEnd
	}

	return output.String()
}

func (c *copyState) addTarget(x, y, width int, rawValue string) {
	if width <= 0 || rawValue == "" {
		return
	}
	c.targets = append(c.targets, copyHitTarget{
		x1:       x,
		x2:       x + width,
		y:        y,
		rawValue: rawValue,
	})
}

func (c *copyState) isHovered(x, y, width int) bool {
	if !c.hasHover {
		return false
	}
	return c.hoverY == y && c.hoverX >= x && c.hoverX < x+width
}

func (c *copyState) renderValue(p palette, x, y int, displayText, rawValue string, baseStyle lipgloss.Style) string {
	width := lipgloss.Width(displayText)
	c.addTarget(x, y, width, rawValue)

	if c.isHovered(x, y, width) {
		hoverStyle := baseStyle.
			Background(p.accent).
			Foreground(p.background).
			Bold(true).
			Underline(true)
		return hoverStyle.Render(displayText)
	}

	return baseStyle.Render(displayText)
}

func (c *copyState) checkClick(x, y int) (string, bool) {
	for _, target := range c.targets {
		if target.y == y && x >= target.x1 && x < target.x2 {
			return target.rawValue, true
		}
	}
	return "", false
}

func (m *model) copyToClipboard(val string) tea.Cmd {
	if val == "" {
		return nil
	}
	if err := clipboard.WriteAll(val); err != nil {
		m.copy.toastSuccess = false
		m.copy.toastText = "Copy failed"
	} else {
		m.copy.toastSuccess = true
		m.copy.toastText = fmt.Sprintf("Copied %s", val)
	}
	m.copy.toastTimerID++
	id := m.copy.toastTimerID

	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearCopyToastMsg{id: id}
	})
}

func (m model) renderCopyToast(width int, p palette) string {
	if m.copy.toastText == "" {
		return ""
	}
	toastStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(p.background).
		Background(p.accent).
		Padding(0, 1)

	marker := "✓ "
	if !m.copy.toastSuccess {
		marker = "! "
	}
	rendered := toastStyle.Render(marker + m.copy.toastText)
	toastWidth := lipgloss.Width(rendered)
	if toastWidth >= width {
		return rendered
	}

	pad := width - toastWidth - 2
	if pad < 0 {
		pad = 0
	}
	return lipgloss.NewStyle().MarginLeft(pad).Render(rendered)
}

func (m model) markCopyValue(rawValue, displayText string) string {
	if m.copy == nil {
		return displayText
	}
	return m.copy.markValue(rawValue, displayText)
}
