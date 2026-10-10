package main

import (
	"fmt"
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
	toastTimerID int
	targets      []copyHitTarget
}

func (c *copyState) clearTargets() {
	c.targets = c.targets[:0]
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
	_ = clipboard.WriteAll(val)
	m.copy.toastTimerID++
	m.copy.toastText = fmt.Sprintf("Copied %s", val)
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

	rendered := toastStyle.Render("✓ " + m.copy.toastText)
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
