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

func pistonDistance(m model) float64 {
	value, err := strconv.ParseFloat(
		strings.TrimSpace(
			m.pistonDistanceInput.Value(),
		),
		64,
	)

	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}

	return value
}

func pistonAmount(m model) int {
	value, err := strconv.Atoi(
		strings.TrimSpace(
			m.pistonAmountInput.Value(),
		),
	)

	if err != nil || value < 1 {
		return 0
	}

	return value
}

func pistonValue(m model) float64 {
	value, err := strconv.ParseFloat(
		strings.TrimSpace(
			m.pistonValueInput.Value(),
		),
		64,
	)

	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}

	return value
}

func calculatePistonLength(
	distance float64,
	pistons int,
	value float64,
) float64 {
	if distance < 0 ||
		pistons < 1 ||
		value < 0 ||
		math.IsNaN(distance) ||
		math.IsInf(distance, 0) ||
		math.IsNaN(value) ||
		math.IsInf(value, 0) {
		return 0
	}

	return (value * distance) / float64(pistons)
}

func pistonLengthValid(
	distance float64,
	pistons int,
	value float64,
) bool {
	return distance >= 0 &&
		pistons >= 1 &&
		value >= 0 &&
		!math.IsNaN(distance) &&
		!math.IsInf(distance, 0) &&
		!math.IsNaN(value) &&
		!math.IsInf(value, 0)
}

func (m model) updatePiston(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.pistonDistanceInput.Blur()
		m.pistonAmountInput.Blur()
		m.pistonValueInput.Blur()
		m.page = pageHome
		return m, nil

	case "c":
		m.pistonDistanceInput.Blur()
		m.pistonAmountInput.Blur()
		m.pistonValueInput.Blur()
		m.customizeTool = customizePiston
		m.customIndex = 0
		m.page = pageCustomize
		return m, nil

	case "tab":
		if m.pistonDistanceInput.Focused() {
			m.pistonDistanceInput.Blur()
			m.pistonAmountInput.Focus()
			m.pistonAmountInput.CursorEnd()
			m.pistonFieldIndex = 1
			return m, textinput.Blink
		}

		if m.pistonAmountInput.Focused() {
			m.pistonAmountInput.Blur()
			m.pistonValueInput.Focus()
			m.pistonValueInput.CursorEnd()
			m.pistonFieldIndex = 2
			return m, textinput.Blink
		}

		m.pistonValueInput.Blur()
		m.pistonDistanceInput.Focus()
		m.pistonDistanceInput.CursorEnd()
		m.pistonFieldIndex = 0
		return m, textinput.Blink

	case "shift+tab":
		if m.pistonValueInput.Focused() {
			m.pistonValueInput.Blur()
			m.pistonAmountInput.Focus()
			m.pistonAmountInput.CursorEnd()
			m.pistonFieldIndex = 1
			return m, textinput.Blink
		}

		if m.pistonAmountInput.Focused() {
			m.pistonAmountInput.Blur()
			m.pistonDistanceInput.Focus()
			m.pistonDistanceInput.CursorEnd()
			m.pistonFieldIndex = 0
			return m, textinput.Blink
		}

		m.pistonDistanceInput.Blur()
		m.pistonValueInput.Focus()
		m.pistonValueInput.CursorEnd()
		m.pistonFieldIndex = 2
		return m, textinput.Blink

	case "enter", "e":
		if m.pistonDistanceInput.Focused() {
			m.pistonDistanceInput.Blur()
			return m, nil
		}

		if m.pistonAmountInput.Focused() {
			m.pistonAmountInput.Blur()
			return m, nil
		}

		if m.pistonValueInput.Focused() {
			m.pistonValueInput.Blur()
			return m, nil
		}

		if m.pistonFieldIndex == 1 {
			m.pistonAmountInput.Focus()
			m.pistonAmountInput.CursorEnd()
		} else {
			m.pistonDistanceInput.Focus()
			m.pistonDistanceInput.CursorEnd()
		}

		return m, textinput.Blink
	}

	if m.pistonDistanceInput.Focused() {
		var cmd tea.Cmd

		m.pistonDistanceInput, cmd =
			m.pistonDistanceInput.Update(msg)

		return m, cmd
	}

	if m.pistonAmountInput.Focused() {
		var cmd tea.Cmd

		m.pistonAmountInput, cmd =
			m.pistonAmountInput.Update(msg)

		return m, cmd
	}

	if m.pistonValueInput.Focused() {
		var cmd tea.Cmd

		m.pistonValueInput, cmd =
			m.pistonValueInput.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m model) handlePistonMouse(
	msg tea.MouseClickMsg,
) (tea.Model, tea.Cmd) {

	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	width := m.width

	if width < 1 {
		width = 1
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

	inputX := 0

	if m.cfg.Piston.ResultSide == "left" {
		inputX = leftWidth + 1
	}

	const fieldWidth = 12

	fieldX1 := inputX + 2
	fieldX2 := fieldX1 + fieldWidth

	contentY := msg.Y - 2

	// CRANK DISTANCE field.
	if (contentY == 3 || contentY == 4) &&
		msg.X >= fieldX1 &&
		msg.X < fieldX2 {

		m.pistonAmountInput.Blur()
		m.pistonValueInput.Blur()

		m.pistonDistanceInput.Focus()
		m.pistonDistanceInput.CursorEnd()
		m.pistonFieldIndex = 0

		return m, textinput.Blink
	}

	// AMOUNT OF PISTONS field.
	if (contentY == 6 || contentY == 7) &&
		msg.X >= fieldX1 &&
		msg.X < fieldX2 {

		m.pistonDistanceInput.Blur()
		m.pistonValueInput.Blur()

		m.pistonAmountInput.Focus()
		m.pistonAmountInput.CursorEnd()
		m.pistonFieldIndex = 1

		return m, textinput.Blink
	}

	// VALUE field.
	if (contentY == 9 || contentY == 10) &&
		msg.X >= fieldX1 &&
		msg.X < fieldX2 {

		m.pistonDistanceInput.Blur()
		m.pistonAmountInput.Blur()

		m.pistonValueInput.Focus()
		m.pistonValueInput.CursorEnd()
		m.pistonFieldIndex = 2

		return m, textinput.Blink
	}

	m.pistonDistanceInput.Blur()
	m.pistonAmountInput.Blur()
	m.pistonValueInput.Blur()

	return m, nil
}

func (m model) viewPiston() tea.View {

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
			Width(12).
			Foreground(p.text).
			Padding(0, 0)

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

	distanceField :=
		inputStyle.Render(
			m.pistonDistanceInput.Value(),
		)

	if m.pistonDistanceInput.Focused() {
		distanceField =
			focusedInputStyle.Render(
				m.pistonDistanceInput.View(),
			)
	}

	amountField :=
		inputStyle.Render(
			m.pistonAmountInput.Value(),
		)

	if m.pistonAmountInput.Focused() {
		amountField =
			focusedInputStyle.Render(
				m.pistonAmountInput.View(),
			)
	}

	valueField :=
		inputStyle.Render(
			m.pistonValueInput.Value(),
		)

	if m.pistonValueInput.Focused() {
		valueField =
			focusedInputStyle.Render(
				m.pistonValueInput.View(),
			)
	}

	leftContent :=
		strings.Join(
			[]string{
				titleStyle.Render(
					"PISTON LENGTH CALCULATOR",
				),
				"",
				labelStyle.Render(
					"CRANK DISTANCE",
				),
				distanceField,
				"",
				labelStyle.Render(
					"AMOUNT OF PISTONS",
				),
				amountField,
				"",
				labelStyle.Render(
					"VALUE",
				),
				valueField,
				"",
				labelStyle.Render(
					"[TAB] Next field    [ENTER/E] Edit",
				),
				labelStyle.Render(
					"[C] Customize    [ESC] Back",
				),
			},
			"\n",
		)

	resultLines :=
		[]string{
			titleStyle.Render(
				"PISTON LENGTH",
			),
			"",
		}

	distanceText :=
		strings.TrimSpace(
			m.pistonDistanceInput.Value(),
		)

	pistonsText :=
		strings.TrimSpace(
			m.pistonAmountInput.Value(),
		)

	distanceValue, distanceErr :=
		strconv.ParseFloat(
			distanceText,
			64,
		)

	pistonsValue, pistonsErr :=
		strconv.Atoi(
			pistonsText,
		)

	valueText :=
		strings.TrimSpace(
			m.pistonValueInput.Value(),
		)

	valueValue, valueErr :=
		strconv.ParseFloat(
			valueText,
			64,
		)

	if distanceErr != nil ||
		math.IsNaN(distanceValue) || math.IsInf(distanceValue, 0) ||
		distanceValue < 0 ||
		pistonsErr != nil ||
		pistonsValue < 1 ||
		valueErr != nil ||
		math.IsNaN(valueValue) || math.IsInf(valueValue, 0) ||
		valueValue < 0 {

		resultLines =
			append(
				resultLines,
				labelStyle.Render(
					"Enter valid values.",
				),
			)

	} else {

		calculated :=
			calculatePistonLength(
				distanceValue,
				pistonsValue,
				valueValue,
			)

		if calculated > 4 {

			resultValueStyle :=
				valueStyle

			resultWarningStyle :=
				lipgloss.NewStyle().
					Bold(true).
					Foreground(
						lipgloss.Color(
							"#f38ba8",
						),
					)

			separatorStyle :=
				lipgloss.NewStyle()

			if m.cfg.Appearance.Background != "transparent" {
				resultValueStyle =
					resultValueStyle.Background(p.panel)

				resultWarningStyle =
					resultWarningStyle.Background(p.panel)

				separatorStyle =
					separatorStyle.Background(p.panel)
			}

			valueText :=
				fmt.Sprintf(
					"%.3f",
					calculated,
				)

			resultLines =
				append(
					resultLines,
					resultValueStyle.Render(valueText)+
						separatorStyle.Render(" ")+
						resultWarningStyle.Render(
							"VALUE EXCEEDS LIMIT",
						),
				)

		} else {

			resultLines =
				append(
					resultLines,
					valueStyle.Render(
						fmt.Sprintf(
							"%.3f",
							calculated,
						),
					),
				)
		}
	}

	rightContent :=
		strings.Join(
			resultLines,
			"\n",
		)

	panelHeight := height - 2

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

	renderPanel :=
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
		renderPanel(
			leftContent,
			leftWidth,
		)

	rightPanelStyle :=
		lipgloss.NewStyle().
			Width(rightWidth).
			Height(panelHeight).
			Padding(1, 2).
			Foreground(p.text)

	if m.cfg.Appearance.Background != "transparent" {
		rightPanelStyle =
			rightPanelStyle.Background(p.panel)
	}

	rightPanel :=
		rightPanelStyle.Render(
			rightContent,
		)

	dividerLines :=
		make(
			[]string,
			panelHeight,
		)

	for i := range dividerLines {

		dividerLines[i] =
			"│"
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

	if m.cfg.Piston.ResultSide == "left" {

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

	titleText :=
		"PC MULTITOOL  •  PISTON LENGTH CALCULATOR"

	authorText :=
		"Made by Xad0"

	titleStyleTop :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	authorStyle :=
		lipgloss.NewStyle().
			Foreground(p.muted)

	if m.cfg.Appearance.Background !=
		"transparent" {

		titleStyleTop =
			titleStyleTop.Background(p.surface)

		authorStyle =
			authorStyle.Background(p.surface)
	}

	title :=
		titleStyleTop.Render(titleText)

	author :=
		authorStyle.Render(authorText)

	gap :=
		width -
			lipgloss.Width(titleText) -
			lipgloss.Width(authorText)

	if gap < 1 {
		gap = 1
	}

	topStyle :=
		lipgloss.NewStyle().
			Width(width).
			Height(2)

	if m.cfg.Appearance.Background !=
		"transparent" {

		topStyle =
			topStyle.Background(p.surface)
	}

	top :=
		topStyle.Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				title,
				strings.Repeat(
					" ",
					gap,
				),
				author,
			),
		)

	content :=
		strings.Join(
			[]string{
				top,
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
