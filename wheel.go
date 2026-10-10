package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type wheelData struct {
	TireSize         int
	AngleLock        float64
	CenterAngle      float64
	CenterAngleValid bool
	Offset           float64
	Compressors      int
	CompressorValue  float64
}

func wheelInput(value int) textinput.Model {
	input := textinput.New()
	input.SetValue(strconv.Itoa(value))
	input.SetWidth(14)
	input.CharLimit = 4
	input.Prompt = ""
	input.Blur()

	return input
}

func wheelCurrentTireSize(m model) int {
	value, err := strconv.Atoi(
		m.wheelTireInput.Value(),
	)

	if err != nil || value <= 0 {
		return 10
	}

	if value > 9999 {
		return 9999
	}

	return value
}

func wheelOffset(tireSize int) float64 {
	if tireSize < 2 {
		return 0
	}

	angleRadians :=
		math.Pi /
			float64(tireSize)

	return 1.0 /
		math.Tan(angleRadians)
}

func wheelCompressors(offset float64) int {
	n :=
		int(math.Floor(offset)) -
			1

	if n < 1 {
		return 1
	}

	return n
}

func wheelCompressorValue(
	offset float64,
	compressors int,
) float64 {
	value :=
		offset -
			float64(compressors) -
			1.0

	if value < 0 {
		return 0
	}

	return value
}

func calculateWheel(size int) wheelData {
	if size <= 0 {
		return wheelData{}
	}

	angleLock :=
		360.0 /
			float64(size)

	centerAngle := 0.0
	centerAngleValid := size%2 == 0

	if centerAngleValid {
		centerAngle =
			angleLock / 2.0
	}

	offset :=
		wheelOffset(size)

	compressors :=
		wheelCompressors(offset)

	compressorValue :=
		Round3(
			wheelCompressorValue(
				offset,
				compressors,
			),
		)

	return wheelData{
		TireSize:         size,
		AngleLock:        angleLock,
		CenterAngle:      centerAngle,
		CenterAngleValid: centerAngleValid,
		Offset:           offset,
		Compressors:      compressors,
		CompressorValue:  compressorValue,
	}
}

func (m model) updateWheel(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.wheelTireInput.Blur()
		m.page = pageHome
		return m, nil

	case "c":
		m.wheelTireInput.Blur()
		m.customizeTool = customizeWheel
		m.customIndex = 0
		m.page = pageCustomize
		return m, nil

	case "enter", "e":
		if !m.wheelTireInput.Focused() {
			m.wheelTireInput.Focus()
			m.wheelTireInput.CursorEnd()

			return m, textinput.Blink
		}

		m.wheelTireInput.Blur()
		return m, nil
	}

	if m.wheelTireInput.Focused() {
		var cmd tea.Cmd

		m.wheelTireInput, cmd =
			m.wheelTireInput.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m model) handleWheelMouse(
	msg tea.MouseClickMsg,
) (tea.Model, tea.Cmd) {

	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	width := m.width
	height := m.height

	if width < 1 {
		width = 1
	}

	if height < 1 {
		height = 1
	}

	panelHeight := height - 2
	if panelHeight < 1 {
		panelHeight = 1
	}

	leftWidth := width / 2

	if leftWidth < 30 {
		leftWidth = 30
	}

	if leftWidth > width-20 {
		leftWidth = width - 20
	}

	if leftWidth < 1 {
		leftWidth = 1
	}

	rightWidth := width - leftWidth - 1
	if rightWidth < 1 {
		rightWidth = 1
	}

	// The input panel is normally on the left.
	// When results are configured on the left, the input
	// panel moves to the right.
	inputX := 0

	if m.cfg.Wheel.ResultSide == "left" {
		inputX = leftWidth + 1
	}

	contentY := msg.Y - 2

	if contentY < 0 || contentY >= panelHeight {
		m.wheelTireInput.Blur()
		return m, nil
	}

	if msg.X >= inputX+2 &&
		msg.X < inputX+leftWidth-2 {

		switch contentY {
		case 3, 4:
			m.wheelTireInput.Focus()
			m.wheelTireInput.CursorEnd()
			return m, textinput.Blink
		}
	}

	m.wheelTireInput.Blur()

	return m, nil
}

func (m model) viewWheel() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	width :=
		m.width

	height :=
		m.height

	if width < 1 {
		width = 1
	}

	if height < 1 {
		height = 1
	}

	data :=
		calculateWheel(
			wheelCurrentTireSize(m),
		)

	titleStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	labelStyle :=
		lipgloss.NewStyle().
			Foreground(p.muted)

	valueStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.text)

	inputStyle :=
		lipgloss.NewStyle().
			Width(16).
			Foreground(p.text).
			Padding(0, 1)

	if m.cfg.Appearance.Background !=
		"transparent" {

		inputStyle =
			inputStyle.Background(
				p.surface,
			)
	}

	focusedInputStyle :=
		inputStyle.
			Border(
				lipgloss.NormalBorder(),
			).
			BorderForeground(
				p.accent,
			)

	// ─────────────────────────────────────────────────────────
	// Left panel
	// ─────────────────────────────────────────────────────────

	leftLines :=
		[]string{
			titleStyle.Render(
				"WHEEL CALCULATOR",
			),
			"",
			labelStyle.Render(
				"TIRE SIZE",
			),
		}

	tireField :=
		inputStyle.Render(
			m.wheelTireInput.Value(),
		)

	if m.wheelTireInput.Focused() {

		tireField =
			focusedInputStyle.Render(
				m.wheelTireInput.View(),
			)
	}

	leftLines =
		append(
			leftLines,
			tireField,
			"",
			labelStyle.Render(
				"[ENTER / E] Edit tire size",
			),
			labelStyle.Render(
				"[C] Customize    [ESC] Back",
			),
		)

	leftContent :=
		strings.Join(
			leftLines,
			"\n",
		)

	// ─────────────────────────────────────────────────────────
	// Right panel
	// ─────────────────────────────────────────────────────────

	rightLines :=
		[]string{
			titleStyle.Render(
				fmt.Sprintf(
					"TIRE %d SM",
					data.TireSize,
				),
			),
			"",
			labelStyle.Render(
				"TIRE SIZE",
			),
			valueStyle.Render(
				m.markCopyValue(strconv.Itoa(data.TireSize), fmt.Sprintf("%d sm", data.TireSize)),
			),
			"",
			labelStyle.Render(
				"ANGLE LOCK",
			),
			valueStyle.Render(
				m.markCopyValue(fmt.Sprintf("%.3f", data.AngleLock), fmt.Sprintf("%.3f°", data.AngleLock)),
			),
			"",
			labelStyle.Render(
				"CENTER ANGLE",
			),
		}

	if data.CenterAngleValid {
		rightLines =
			append(
				rightLines,
				valueStyle.Render(
					m.markCopyValue(fmt.Sprintf("%.3f", data.CenterAngle), fmt.Sprintf("%.3f°", data.CenterAngle)),
				),
			)
	} else {
		rightLines =
			append(
				rightLines,
				valueStyle.Render(
					"N/A",
				),
			)
	}

	rightLines =
		append(
			rightLines,
			"",
			labelStyle.Render(
				"OFFSET",
			),
			valueStyle.Render(
				m.markCopyValue(fmt.Sprintf("%.3f", Round3(data.Offset)), fmt.Sprintf("%.3f", Round3(data.Offset))),
			),
			"",
			labelStyle.Render(
				"COMPRESSORS",
			),
			valueStyle.Render(
				m.markCopyValue(strconv.Itoa(data.Compressors), strconv.Itoa(data.Compressors)),
			),
			"",
			labelStyle.Render(
				"COMPRESSOR VALUE",
			),
			valueStyle.Render(
				m.markCopyValue(fmt.Sprintf("%.3f", data.CompressorValue), fmt.Sprintf("%.3f", data.CompressorValue)),
			),
		)

	rightContent :=
		strings.Join(
			rightLines,
			"\n",
		)

	// ─────────────────────────────────────────────────────────
	// Panels
	// ─────────────────────────────────────────────────────────

	panelHeight :=
		height - 2

	if panelHeight < 1 {
		panelHeight = 1
	}

	leftWidth :=
		width / 2

	if leftWidth < 30 {
		leftWidth = 30
	}

	if leftWidth > width-20 {
		leftWidth = width - 20
	}

	if leftWidth < 1 {
		leftWidth = 1
	}

	rightWidth :=
		width -
			leftWidth -
			1

	if rightWidth < 1 {
		rightWidth = 1
	}

	panelStyle :=
		func(
			content string,
			panelWidth int,
		) string {

			style :=
				lipgloss.NewStyle().
					Width(panelWidth).
					Height(panelHeight).
					Padding(1, 2).
					Foreground(p.text)

			if m.cfg.Appearance.Background !=
				"transparent" {

				style =
					style.Background(
						p.panel,
					)
			}

			return style.Render(content)
		}

	leftPanel :=
		panelStyle(
			leftContent,
			leftWidth,
		)

	rightPanel :=
		panelStyle(
			rightContent,
			rightWidth,
		)

	dividerLines :=
		make(
			[]string,
			panelHeight,
		)

	for i := range dividerLines {
		dividerLines[i] = "│"
	}

	divider :=
		lipgloss.NewStyle().
			Width(1).
			Height(panelHeight).
			Foreground(p.border).
			Render(
				strings.Join(
					dividerLines,
					"\n",
				),
			)

	var main string

	if m.cfg.Wheel.ResultSide == "left" {
		main =
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				rightPanel,
				divider,
				leftPanel,
			)
	} else {
		main =
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				leftPanel,
				divider,
				rightPanel,
			)
	}

	// ─────────────────────────────────────────────────────────
	// Top bar
	// ─────────────────────────────────────────────────────────

	wheelTitleText := "PC MULTITOOL  •  WHEEL CALCULATOR"
	wheelAuthorText := "Made by Xad0"

	wheelTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(p.accent)

	wheelAuthorStyle := lipgloss.NewStyle().
		Foreground(p.muted)

	if m.cfg.Appearance.Background != "transparent" {
		wheelTitleStyle =
			wheelTitleStyle.Background(p.surface)
		wheelAuthorStyle =
			wheelAuthorStyle.Background(p.surface)
	}

	wheelTitle := wheelTitleStyle.Render(wheelTitleText)
	wheelAuthor := wheelAuthorStyle.Render(wheelAuthorText)

	wheelGap := width -
		lipgloss.Width(wheelTitleText) -
		lipgloss.Width(wheelAuthorText)

	if wheelGap < 1 {
		wheelGap = 1
	}

	wheelTopBarStyle := lipgloss.NewStyle().
		Width(width).
		Height(2)

	if m.cfg.Appearance.Background != "transparent" {
		wheelTopBarStyle =
			wheelTopBarStyle.Background(p.surface)
	}

	wheelTopBar := wheelTopBarStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			wheelTitle,
			strings.Repeat(" ", wheelGap),
			wheelAuthor,
		),
	)

	content :=
		strings.Join(
			[]string{
				wheelTopBar,
				main,
			},
			"\n",
		)

	view :=
		tea.NewView(content)

	view.AltScreen = true
	view.MouseMode =
		tea.MouseModeCellMotion

	if m.cfg.Appearance.Background !=
		"transparent" {

		view.BackgroundColor =
			p.background
	}

	return view
}
