package main

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type crankLayout int

const (
	crankInline crankLayout = iota
	crankV
	crankBoxer
)

func crankLayoutName(layout crankLayout) string {
	switch layout {
	case crankInline:
		return "Inline"
	case crankV:
		return "V"
	case crankBoxer:
		return "Boxer"
	default:
		return "Unknown"
	}
}

func crankEffectivePositions(layout crankLayout, cylinders int) int {
	switch layout {
	case crankInline:
		return cylinders

	case crankV, crankBoxer:
		if cylinders <= 0 {
			return 0
		}

		return (cylinders + 1) / 2

	default:
		return 0
	}
}

func crankAngle(layout crankLayout, cylinders int) float64 {
	positions := crankEffectivePositions(
		layout,
		cylinders,
	)

	if positions <= 0 {
		return 0
	}

	return 360.0 / float64(positions)
}

func crankInput(value int) textinput.Model {
	input := textinput.New()
	input.SetValue(strconv.Itoa(value))
	input.SetWidth(12)
	input.CharLimit = 3
	input.Prompt = ""
	input.Blur()
	return input
}

func crankCurrentCylinders(m model) int {
	value, err := strconv.Atoi(
		m.crankCylinderInput.Value(),
	)

	if err != nil || value <= 0 {
		return 6
	}

	if value > 999 {
		return 999
	}

	return value
}

func crankValid(layout crankLayout, cylinders int) bool {
	if cylinders <= 0 {
		return false
	}

	switch layout {
	case crankInline:
		return true

	case crankV, crankBoxer:
		return cylinders > 0

	default:
		return false
	}
}

func crankEngineName(layout crankLayout, cylinders int) string {
	switch layout {
	case crankInline:
		return fmt.Sprintf("I%d", cylinders)

	case crankV:
		return fmt.Sprintf("V%d", cylinders)

	case crankBoxer:
		return fmt.Sprintf("Boxer %d", cylinders)

	default:
		return "Unknown"
	}
}

func crankFormulaText(layout crankLayout, cylinders int) string {
	positions := crankEffectivePositions(
		layout,
		cylinders,
	)

	if positions <= 0 {
		return "Invalid configuration"
	}

	return fmt.Sprintf(
		"360° / %d",
		positions,
	)
}

func (m model) updateCrank(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.crankCylinderInput.Blur()
		m.page = pageHome
		return m, nil

	case "c":
		m.crankCylinderInput.Blur()
		m.customizeTool = customizeCrank
		m.customIndex = 0
		m.page = pageCustomize
		return m, nil

	case "up":
		if !m.crankCylinderInput.Focused() {
			m.crankLayoutIndex--

			if m.crankLayoutIndex < 0 {
				m.crankLayoutIndex = 2
			}

			return m, nil
		}

	case "down":
		if !m.crankCylinderInput.Focused() {
			m.crankLayoutIndex++

			if m.crankLayoutIndex > 2 {
				m.crankLayoutIndex = 0
			}

			return m, nil
		}

	case "left":
		if !m.crankCylinderInput.Focused() {
			m.crankLayoutIndex--

			if m.crankLayoutIndex < 0 {
				m.crankLayoutIndex = 2
			}

			return m, nil
		}

	case "right":
		if !m.crankCylinderInput.Focused() {
			m.crankLayoutIndex++

			if m.crankLayoutIndex > 2 {
				m.crankLayoutIndex = 0
			}

			return m, nil
		}

	case "enter", "e":
		if !m.crankCylinderInput.Focused() {
			m.crankCylinderInput.Focus()
			m.crankCylinderInput.CursorEnd()
			return m, textinput.Blink
		}

		m.crankCylinderInput.Blur()
		return m, nil
	}

	if m.crankCylinderInput.Focused() {
		var cmd tea.Cmd

		m.crankCylinderInput, cmd =
			m.crankCylinderInput.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m model) handleCrankMouse(
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

	// The input/control panel is normally on the left.
	// When results are configured on the left, the input panel
	// moves to the right.
	inputX := 0

	if m.cfg.Crank.Layout == "results-left" {
		inputX = leftWidth + 1
	}

	contentY := msg.Y - 2

	if contentY < 0 || contentY >= panelHeight {
		m.crankCylinderInput.Blur()
		return m, nil
	}

	// Keep the same vertical positions used by the rendered
	// control panel.
	if msg.X < inputX+2 ||
		msg.X >= inputX+leftWidth-2 {

		m.crankCylinderInput.Blur()
		return m, nil
	}

	switch contentY {

	case 5:
		m.crankCylinderInput.Blur()
		m.crankLayoutIndex = int(crankInline)
		return m, nil

	case 6:
		m.crankCylinderInput.Blur()
		m.crankLayoutIndex = int(crankV)
		return m, nil

	case 7:
		m.crankCylinderInput.Blur()
		m.crankLayoutIndex = int(crankBoxer)
		return m, nil

	case 9, 10:
		m.crankCylinderInput.Focus()
		m.crankCylinderInput.CursorEnd()
		return m, textinput.Blink
	}

	m.crankCylinderInput.Blur()

	return m, nil
}

func (m model) viewCrank() tea.View {
	p := getPalette(
		m.theme,
		m.accent,
	)

	width := m.width
	height := m.height

	if width < 1 {
		width = 1
	}

	if height < 1 {
		height = 1
	}

	layout :=
		crankLayout(
			m.crankLayoutIndex,
		)

	cylinders :=
		crankCurrentCylinders(m)

	valid :=
		crankValid(
			layout,
			cylinders,
		)

	positions := 0
	angle := 0.0

	if valid {
		positions =
			crankEffectivePositions(
				layout,
				cylinders,
			)

		angle =
			crankAngle(
				layout,
				cylinders,
			)
	}

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

	selectedStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	normalStyle :=
		lipgloss.NewStyle().
			Foreground(p.text)

	inputStyle :=
		lipgloss.NewStyle().
			Width(16).
			Foreground(p.text).
			Padding(0, 1)

	if m.cfg.Appearance.Background != "transparent" {
		inputStyle =
			inputStyle.Background(p.surface)
	}

	focusedInputStyle :=
		inputStyle.
			Border(lipgloss.NormalBorder()).
			BorderForeground(p.accent)

	leftLines := []string{
		titleStyle.Render(
			"CRANK ANGLE CALCULATOR",
		),
		"",
		labelStyle.Render("LAYOUT"),
		"",
	}

	layouts := []crankLayout{
		crankInline,
		crankV,
		crankBoxer,
	}

	for _, option := range layouts {

		prefix := "  "
		style := normalStyle

		if option == layout {
			prefix = "> "
			style = selectedStyle
		}

		leftLines =
			append(
				leftLines,
				style.Render(
					prefix+crankLayoutName(option),
				),
			)
	}

	leftLines =
		append(
			leftLines,
			"",
			labelStyle.Render("CYLINDERS"),
		)

	cylinderField :=
		inputStyle.Render(
			m.crankCylinderInput.Value(),
		)

	if m.crankCylinderInput.Focused() {
		cylinderField =
			focusedInputStyle.Render(
				m.crankCylinderInput.View(),
			)
	}

	leftLines =
		append(
			leftLines,
			cylinderField,
			"",
			labelStyle.Render(
				"[↑↓ / ←→] Layout",
			),
			labelStyle.Render(
				"[ENTER / E] Edit cylinders",
			),
			labelStyle.Render(
				"[C] Customize    [ESC] Back",
			),
		)

	rightLines := []string{
		titleStyle.Render(
			crankEngineName(
				layout,
				cylinders,
			),
		),
		"",
		labelStyle.Render("LAYOUT"),
		valueStyle.Render(
			crankLayoutName(layout),
		),
		"",
		labelStyle.Render("CYLINDERS"),
		valueStyle.Render(
			strconv.Itoa(cylinders),
		),
		"",
	}

	if !valid {

		rightLines =
			append(
				rightLines,
				labelStyle.Render("STATUS"),
				lipgloss.NewStyle().
					Bold(true).
					Foreground(
						lipgloss.Color("#f38ba8"),
					).
					Render(
						"Cylinder count must be positive.",
					),
			)

	} else {

		rightLines =
			append(
				rightLines,
				labelStyle.Render(
					"BASE PHASE SPACING",
				),
				valueStyle.Render(
					fmt.Sprintf(
						"%.3f°",
						angle,
					),
				),
				"",
				labelStyle.Render(
					"EFFECTIVE POSITIONS",
				),
				valueStyle.Render(
					strconv.Itoa(positions),
				),
				"",
				labelStyle.Render("FORMULA"),
				valueStyle.Render(
					crankFormulaText(
						layout,
						cylinders,
					),
				),
				"",
				labelStyle.Render("PHASE POSITIONS"),
				"",
			)

		for i := 0; i < positions; i++ {

			phase :=
				float64(i) * angle

			rightLines =
				append(
					rightLines,
					fmt.Sprintf(
						"Position %-2d  %9.3f°",
						i+1,
						phase,
					),
				)
		}
	}

	leftContent :=
		strings.Join(
			leftLines,
			"\n",
		)

	rightContent :=
		strings.Join(
			rightLines,
			"\n",
		)

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

	rightWidth :=
		width - leftWidth - 1

	if rightWidth < 1 {
		rightWidth = 1
	}

	panelStyle := func(
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
				style.Background(p.panel)
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

	if m.cfg.Crank.Layout == "results-left" {
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

	crankTitleText := "PC MULTITOOL  •  CRANK ANGLE CALCULATOR"
	crankAuthorText := "Made by Xad0"

	crankTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(p.accent)

	crankAuthorStyle := lipgloss.NewStyle().
		Foreground(p.muted)

	if m.cfg.Appearance.Background != "transparent" {
		crankTitleStyle =
			crankTitleStyle.Background(p.surface)
		crankAuthorStyle =
			crankAuthorStyle.Background(p.surface)
	}

	crankTitle := crankTitleStyle.Render(crankTitleText)
	crankAuthor := crankAuthorStyle.Render(crankAuthorText)

	crankGap := width -
		lipgloss.Width(crankTitleText) -
		lipgloss.Width(crankAuthorText)

	if crankGap < 1 {
		crankGap = 1
	}

	crankTopBarStyle := lipgloss.NewStyle().
		Width(width).
		Height(2)

	if m.cfg.Appearance.Background != "transparent" {
		crankTopBarStyle =
			crankTopBarStyle.Background(p.surface)
	}

	crankTopBar := crankTopBarStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			crankTitle,
			strings.Repeat(" ", crankGap),
			crankAuthor,
		),
	)

	content :=
		strings.Join(
			[]string{
				crankTopBar,
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
