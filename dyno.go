package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const dynoInitialRows = 5

const (
	dynoFieldSPS = iota
	dynoFieldTorque
)

const (
	dynoVisibleRows = 5

	dynoNumberWidth = 5
	dynoFieldWidth  = 10
	dynoGapWidth    = 3

	dynoSPSX        = 9
	dynoTorqueX     = 22
	dynoRemoveX     = 32
	dynoRemoveWidth = 3

	dynoFirstRowY = 7
	dynoRowStep   = 2
)

type dynoPoint struct {
	SPSInput    textinput.Model
	TorqueInput textinput.Model
}

type dynoGraphMode int

const (
	dynoGraphBHP dynoGraphMode = iota
	dynoGraphTorque
)

func dynoGraphModeTitle(mode dynoGraphMode) string {
	if mode == dynoGraphTorque {
		return "Torque vs SPS"
	}
	return "BHP vs RPM"
}

func dynoGraphXAxisLabel(mode dynoGraphMode) string {
	if mode == dynoGraphTorque {
		return "SPS"
	}
	return "RPM"
}

func dynoGraphYAxisLabel(mode dynoGraphMode) string {
	if mode == dynoGraphTorque {
		return "Torque"
	}
	return "BHP"
}

type dynoGraphPoint struct {
	SPS    float64
	RPM    float64
	Torque float64
	BHP    float64
}

// ─────────────────────────────────────────────────────────────
// Inputs
// ─────────────────────────────────────────────────────────────

func newDynoInput() textinput.Model {
	input := textinput.New()
	input.SetWidth(9)
	input.CharLimit = 12
	input.Prompt = ""
	input.Blur()

	return input
}

func newDynoPoint() dynoPoint {
	return dynoPoint{
		SPSInput:    newDynoInput(),
		TorqueInput: newDynoInput(),
	}
}

func initialDynoPoints() []dynoPoint {
	points := make([]dynoPoint, dynoInitialRows)

	for i := range points {
		points[i] = newDynoPoint()
	}

	return points
}

func (m model) dynoVisibleCount() int {
	if len(m.dynoPoints) < dynoVisibleRows {
		return len(m.dynoPoints)
	}

	return dynoVisibleRows
}

// ─────────────────────────────────────────────────────────────
// Calculations
// ─────────────────────────────────────────────────────────────

func dynoParseFloat(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}

	return number, true
}

func dynoRPM(sps float64) float64 {
	return sps * 3.84
}

func dynoBHP(torque, rpm float64) float64 {
	return torque * rpm / 52.52
}

func (m model) dynoGraphPoints() []dynoGraphPoint {
	points := []dynoGraphPoint{{
		SPS:    0,
		RPM:    0,
		Torque: 0,
		BHP:    0,
	}}

	for i := range m.dynoPoints {
		sps, spsOK :=
			dynoParseFloat(
				m.dynoPoints[i].
					SPSInput.
					Value(),
			)

		torque, torqueOK :=
			dynoParseFloat(
				m.dynoPoints[i].
					TorqueInput.
					Value(),
			)

		if !spsOK || !torqueOK {
			continue
		}

		if sps < 0 {
			continue
		}

		rpm := dynoRPM(sps)

		points = append(
			points,
			dynoGraphPoint{
				SPS:    sps,
				RPM:    rpm,
				Torque: torque,
				BHP:    dynoBHP(torque, rpm),
			},
		)
	}

	// The graph must always progress from low RPM to high RPM.
	sort.SliceStable(
		points,
		func(i, j int) bool {
			return points[i].RPM < points[j].RPM
		},
	)

	return points
}

func dynoGraphSeries(points []dynoGraphPoint, mode dynoGraphMode) []dynoGraphPoint {
	if mode != dynoGraphTorque {
		return points
	}
	series := make([]dynoGraphPoint, len(points))
	for i, point := range points {
		point.RPM = point.SPS
		point.BHP = point.Torque
		series[i] = point
	}
	return series
}

func dynoPeakTorque(
	points []dynoGraphPoint,
) (float64, float64, bool) {
	if len(points) < 2 {
		return 0, 0, false
	}

	peak := points[1]

	for _, point := range points[2:] {
		if point.Torque > peak.Torque {
			peak = point
		}
	}

	return peak.Torque, peak.SPS, true
}

func dynoPeakSPS(
	points []dynoGraphPoint,
) (float64, float64, bool) {
	if len(points) < 2 {
		return 0, 0, false
	}

	peak := points[1]

	for _, point := range points[2:] {
		if point.SPS > peak.SPS {
			peak = point
		}
	}

	return peak.SPS, peak.Torque, true
}

func dynoMeasuredPoints(points []dynoGraphPoint) []dynoGraphPoint {
	if len(points) <= 1 {
		return nil
	}
	return points[1:]
}

func dynoPeakBHP(
	points []dynoGraphPoint,
) (float64, float64, bool) {
	if len(points) < 2 {
		return 0, 0, false
	}

	peak := points[1]

	for _, point := range points[2:] {
		if point.BHP > peak.BHP {
			peak = point
		}
	}

	return peak.BHP, peak.SPS, true
}

// ─────────────────────────────────────────────────────────────
// Graph bounds
// ─────────────────────────────────────────────────────────────

func dynoGraphBounds(
	points []dynoGraphPoint,
) (float64, float64, float64, float64) {

	if len(points) == 0 {
		return 0, 1000, 0, 100
	}

	minRPM := points[0].RPM
	maxRPM := points[0].RPM
	minBHP := points[0].BHP
	maxBHP := points[0].BHP

	for _, point := range points {
		if point.RPM < minRPM {
			minRPM = point.RPM
		}

		if point.RPM > maxRPM {
			maxRPM = point.RPM
		}

		if point.BHP < minBHP {
			minBHP = point.BHP
		}

		if point.BHP > maxBHP {
			maxBHP = point.BHP
		}
	}

	if minBHP > 0 {
		minBHP = 0
	}

	if maxRPM <= minRPM {
		maxRPM = minRPM + 100
	}

	if maxBHP <= minBHP {
		maxBHP = minBHP + 100
	}

	return minRPM,
		maxRPM,
		minBHP,
		maxBHP
}

// ─────────────────────────────────────────────────────────────
// Field state
// ─────────────────────────────────────────────────────────────

func (m model) dynoFieldCount() int {
	return len(m.dynoPoints) * 2
}

func (m model) dynoFieldRow() int {
	if m.dynoFieldIndex < 0 {
		return 0
	}

	return m.dynoFieldIndex / 2
}

func (m model) dynoFieldType() int {
	if m.dynoFieldIndex%2 == 0 {
		return dynoFieldSPS
	}

	return dynoFieldTorque
}

func (m model) dynoFocused() bool {
	row := m.dynoFieldRow()

	if row < 0 ||
		row >= len(m.dynoPoints) {
		return false
	}

	if m.dynoFieldType() == dynoFieldSPS {
		return m.dynoPoints[row].
			SPSInput.
			Focused()
	}

	return m.dynoPoints[row].
		TorqueInput.
		Focused()
}

func (m *model) dynoBlurAll() {
	for i := range m.dynoPoints {
		m.dynoPoints[i].
			SPSInput.
			Blur()

		m.dynoPoints[i].
			TorqueInput.
			Blur()
	}
}

func (m *model) dynoFocusField(
	index int,
) tea.Cmd {

	if len(m.dynoPoints) == 0 {
		return nil
	}

	max :=
		m.dynoFieldCount() -
			1

	if index < 0 {
		index = 0
	}

	if index > max {
		index = max
	}

	m.dynoFieldIndex =
		index

	m.dynoBlurAll()

	row :=
		index / 2

	if index%2 == 0 {

		m.dynoPoints[row].
			SPSInput.
			Focus()

		m.dynoPoints[row].
			SPSInput.
			CursorEnd()

	} else {

		m.dynoPoints[row].
			TorqueInput.
			Focus()

		m.dynoPoints[row].
			TorqueInput.
			CursorEnd()
	}

	m.dynoEnsureVisible()

	return textinput.Blink
}

func (m *model) dynoEnsureVisible() {
	row :=
		m.dynoFieldRow()

	maxScroll :=
		len(m.dynoPoints) -
			dynoVisibleRows

	if maxScroll < 0 {
		maxScroll = 0
	}

	if row < m.dynoScroll {
		m.dynoScroll =
			row
	}

	if row >=
		m.dynoScroll+
			dynoVisibleRows {

		m.dynoScroll =
			row -
				dynoVisibleRows +
				1
	}

	if m.dynoScroll < 0 {
		m.dynoScroll = 0
	}

	if m.dynoScroll > maxScroll {
		m.dynoScroll =
			maxScroll
	}
}

func dynoRemovePoint(points []dynoPoint, index int) ([]dynoPoint, bool) {
	if len(points) <= 1 || index < 0 || index >= len(points) {
		return points, false
	}
	remaining := make([]dynoPoint, 0, len(points)-1)
	remaining = append(remaining, points[:index]...)
	remaining = append(remaining, points[index+1:]...)
	return remaining, true
}

func (m *model) dynoRemoveRow(index int) tea.Cmd {
	if len(m.dynoPoints) <= 1 {
		return nil
	}
	wasFocused := m.dynoFocused()
	oldField := m.dynoFieldIndex
	if wasFocused {
		m.dynoBlurAll()
	}
	remaining, removed := dynoRemovePoint(m.dynoPoints, index)
	if !removed {
		return nil
	}
	m.dynoPoints = remaining
	if wasFocused {
		focusRow := oldField / 2
		if focusRow > index {
			oldField -= 2
		} else if focusRow == index {
			if focusRow >= len(m.dynoPoints) {
				focusRow = len(m.dynoPoints) - 1
			}
			oldField = focusRow * 2
		}
		return m.dynoFocusField(oldField)
	}
	if m.dynoFieldIndex >= m.dynoFieldCount() {
		m.dynoFieldIndex = m.dynoFieldCount() - 1
	}
	m.dynoEnsureVisible()
	return nil
}

func (m *model) dynoAddRow() {
	m.dynoPoints =
		append(
			m.dynoPoints,
			newDynoPoint(),
		)
}

func (m *model) dynoScrollBy(
	delta int,
) {

	maxScroll :=
		len(m.dynoPoints) -
			dynoVisibleRows

	if maxScroll < 0 {
		maxScroll = 0
	}

	m.dynoScroll +=
		delta

	if m.dynoScroll < 0 {
		m.dynoScroll = 0
	}

	if m.dynoScroll > maxScroll {
		m.dynoScroll =
			maxScroll
	}
}

func (m *model) dynoMoveField(
	delta int,
) tea.Cmd {

	if len(m.dynoPoints) == 0 {
		return nil
	}

	next :=
		m.dynoFieldIndex +
			delta

	max :=
		m.dynoFieldCount() -
			1

	if next < 0 {
		next = 0
	}

	if next > max {
		next = max
	}

	return m.dynoFocusField(next)
}

// ─────────────────────────────────────────────────────────────
// Keyboard
// ─────────────────────────────────────────────────────────────

func (m model) updateDyno(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	if len(m.dynoPoints) == 0 {
		m.dynoPoints =
			initialDynoPoints()
	}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "c":
		m.dynoBlurAll()
		m.customizeTool = customizeDyno
		m.customIndex = 0
		m.page = pageCustomize
		return m, nil

	case "esc":

		if m.dynoFullscreen {
			m.dynoFullscreen =
				false

			return m, nil
		}

		m.dynoBlurAll()
		m.page =
			pageHome

		return m, nil

	case "m":
		if m.dynoMode == dynoGraphTorque {
			m.dynoMode = dynoGraphBHP
		} else {
			m.dynoMode = dynoGraphTorque
		}
		return m, nil

	case "g":

		m.dynoBlurAll()

		m.dynoFullscreen =
			!m.dynoFullscreen

		return m, nil

	case "e":

		dynoExportPNG(m)

		return m, nil

	case "tab":

		if m.dynoFocused() {
			return m,
				m.dynoMoveField(1)
		}

	case "shift+tab":

		if m.dynoFocused() {
			return m,
				m.dynoMoveField(-1)
		}

	case "up":

		if m.dynoFocused() {
			return m,
				m.dynoMoveField(-2)
		}

	case "down":

		if m.dynoFocused() {
			return m,
				m.dynoMoveField(2)
		}

	case "left":

		if m.dynoFocused() &&
			m.dynoFieldType() ==
				dynoFieldTorque {

			return m,
				m.dynoMoveField(-1)
		}

	case "right":

		if m.dynoFocused() &&
			m.dynoFieldType() ==
				dynoFieldSPS {

			return m,
				m.dynoMoveField(1)
		}

	case "enter":

		if m.dynoFocused() {
			return m,
				m.dynoMoveField(1)
		}

	case "pageup":

		m.dynoScrollBy(-1)
		return m, nil

	case "pagedown":

		m.dynoScrollBy(1)
		return m, nil
	}

	if !m.dynoFocused() {
		return m, nil
	}

	row :=
		m.dynoFieldRow()

	if row < 0 ||
		row >= len(m.dynoPoints) {
		return m, nil
	}

	var cmd tea.Cmd

	if m.dynoFieldType() ==
		dynoFieldSPS {

		m.dynoPoints[row].
			SPSInput,
			cmd =
			m.dynoPoints[row].
				SPSInput.
				Update(msg)

	} else {

		m.dynoPoints[row].
			TorqueInput,
			cmd =
			m.dynoPoints[row].
				TorqueInput.
				Update(msg)
	}

	return m, cmd
}

// ─────────────────────────────────────────────────────────────
// Mouse
// ─────────────────────────────────────────────────────────────

func (m model) handleDynoMouse(
	msg tea.MouseClickMsg,
) (tea.Model, tea.Cmd) {

	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	width := m.width

	if width < 1 {
		width = 1
	}

	leftWidth := width * 32 / 100

	if leftWidth < 36 {
		leftWidth = 36
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

	graphPanelX := 0
	if !m.dynoFullscreen && m.cfg.Dyno.GraphSide != "left" {
		graphPanelX = leftWidth + 1
	}
	modeX := graphPanelX + 2
	if msg.Y == 3 {
		if msg.X >= modeX && msg.X < modeX+12 {
			m.dynoMode = dynoGraphBHP
			return m, nil
		}
		if msg.X >= modeX+13 && msg.X < modeX+28 {
			m.dynoMode = dynoGraphTorque
			return m, nil
		}
	}
	if m.dynoFullscreen {
		return m, nil
	}

	// The data panel is normally on the left.
	// When the graph is configured on the left, the data
	// panel moves to the right.
	dataPanelX := 0

	if m.cfg.Dyno.GraphSide == "left" {
		dataPanelX = rightWidth + 1
	}

	const (
		firstRowY = dynoFirstRowY
		rowStep   = dynoRowStep
	)

	spsX1 := dataPanelX + dynoSPSX
	spsX2 := dataPanelX + dynoSPSX + dynoFieldWidth

	torqueX1 := dataPanelX + dynoTorqueX
	torqueX2 := dataPanelX + dynoTorqueX + dynoFieldWidth

	visible := m.dynoVisibleCount()

	for visibleRow := 0; visibleRow < visible; visibleRow++ {

		screenY := firstRowY + visibleRow*rowStep

		if msg.Y != screenY {
			continue
		}

		row := m.dynoScroll + visibleRow

		if row < 0 || row >= len(m.dynoPoints) {
			return m, nil
		}

		removeX1 := dataPanelX + dynoRemoveX
		if msg.X >= removeX1 && msg.X < removeX1+dynoRemoveWidth {
			return m, m.dynoRemoveRow(row)
		}

		if msg.X >= spsX1 && msg.X < spsX2 {
			return m, m.dynoFocusField(row * 2)
		}

		if msg.X >= torqueX1 && msg.X < torqueX2 {
			return m, m.dynoFocusField(row*2 + 1)
		}

		m.dynoBlurAll()
		return m, nil
	}

	// Add Row button is also inside the data panel.
	buttonY := firstRowY + visible*rowStep

	if msg.Y == buttonY &&
		msg.X >= dataPanelX+5 &&
		msg.X <= dataPanelX+19 {

		m.dynoBlurAll()
		m.dynoAddRow()
		return m, nil
	}

	m.dynoBlurAll()

	return m, nil
}

func (m model) handleDynoWheel(
	msg tea.MouseWheelMsg,
) (tea.Model, tea.Cmd) {

	if m.dynoFullscreen {
		return m, nil
	}

	mouse :=
		msg.Mouse()

	width :=
		m.width

	if width < 1 {
		width = 1
	}

	leftWidth :=
		width * 32 / 100

	if leftWidth < 36 {
		leftWidth = 36
	}
	if leftWidth > width-20 {
		leftWidth = width - 20
	}
	if leftWidth < 1 {
		leftWidth = 1
	}

	if mouse.X < 0 ||
		mouse.X >= leftWidth {

		return m, nil
	}

	if mouse.Y < 2 {
		return m, nil
	}

	switch msg.Button {

	case tea.MouseWheelUp:
		m.dynoScrollBy(-1)

	case tea.MouseWheelDown:
		m.dynoScrollBy(1)
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// Input rendering
// ─────────────────────────────────────────────────────────────

func (m model) dynoInputView(
	input textinput.Model,
	focused bool,
	p palette,
) string {

	style :=
		lipgloss.NewStyle().
			Width(dynoFieldWidth).
			Height(1).
			Foreground(p.text).
			Background(p.surface).
			Padding(0, 1)

	if focused {

		return style.
			Foreground(p.accent).
			Render(input.View())
	}

	return style.Render(
		input.Value(),
	)
}

// ─────────────────────────────────────────────────────────────
// Data panel
// ─────────────────────────────────────────────────────────────

func (m model) dynoDataView(
	p palette,
) string {

	const tableWidth = 37

	panelBG :=
		p.panel

	lineStyle :=
		lipgloss.NewStyle().
			Width(tableWidth).
			Height(1).
			Background(panelBG)

	titleStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent).
			Background(panelBG)

	mutedStyle :=
		lipgloss.NewStyle().
			Foreground(p.muted).
			Background(panelBG)

	header :=
		lipgloss.JoinHorizontal(
			lipgloss.Left,

			lipgloss.NewStyle().
				Width(7).
				Height(1).
				Background(panelBG).
				Render(""),

			lipgloss.NewStyle().
				Width(10).
				Height(1).
				Align(lipgloss.Center).
				Bold(true).
				Foreground(p.text).
				Background(panelBG).
				Render("SPS"),

			lipgloss.NewStyle().
				Width(3).
				Height(1).
				Background(panelBG).
				Render(""),

			lipgloss.NewStyle().
				Width(10).
				Height(1).
				Align(lipgloss.Center).
				Bold(true).
				Foreground(p.text).
				Background(panelBG).
				Render("TORQUE"),
		)

	lines :=
		[]string{
			lineStyle.Render(
				titleStyle.Render(
					"DYNO DATA",
				),
			),
			lineStyle.Render(""),
			lineStyle.Render(header),
			lineStyle.Render(
				mutedStyle.Render(
					strings.Repeat(
						"─",
						tableWidth,
					),
				),
			),
		}

	start :=
		m.dynoScroll

	if start < 0 {
		start = 0
	}

	end :=
		start +
			m.dynoVisibleCount()

	if end > len(m.dynoPoints) {
		end =
			len(m.dynoPoints)
	}

	for i := start; i < end; i++ {

		number :=
			lipgloss.NewStyle().
				Width(7).
				Height(1).
				Foreground(p.muted).
				Background(panelBG).
				Render(
					fmt.Sprintf(
						"%2d  ",
						i+1,
					),
				)

		sps :=
			m.dynoInputView(
				m.dynoPoints[i].
					SPSInput,
				m.dynoPoints[i].
					SPSInput.
					Focused(),
				p,
			)

		torque :=
			m.dynoInputView(
				m.dynoPoints[i].
					TorqueInput,
				m.dynoPoints[i].
					TorqueInput.
					Focused(),
				p,
			)

		gap :=
			lipgloss.NewStyle().
				Width(dynoGapWidth).
				Height(1).
				Background(panelBG).
				Render("")

		removeStyle := lipgloss.NewStyle().Width(dynoRemoveWidth).Height(1).Align(lipgloss.Center).Bold(true).Foreground(p.accent).Background(panelBG)
		if len(m.dynoPoints) <= 1 {
			removeStyle = removeStyle.Foreground(p.muted)
		}
		removeButton := removeStyle.Render("×")

		row :=
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				number,
				sps,
				gap,
				torque,
				removeButton,
			)

		lines =
			append(
				lines,
				lineStyle.Render(row),
			)

		if i != end-1 {

			lines =
				append(
					lines,
					lineStyle.Render(
						mutedStyle.Render(
							strings.Repeat(
								"─",
								tableWidth,
							),
						),
					),
				)
		}
	}

	lines =
		append(
			lines,
			lineStyle.Render(""),
		)

	button :=
		lipgloss.NewStyle().
			Width(15).
			Height(1).
			Align(lipgloss.Center).
			Bold(true).
			Foreground(p.accent).
			Background(p.surface).
			Render(
				"+ ADD ROW",
			)

	lines =
		append(
			lines,
			lineStyle.Render(button),
		)

	if len(m.dynoPoints) >
		dynoVisibleRows {

		lines =
			append(
				lines,
				lineStyle.Render(""),
				lineStyle.Render(
					mutedStyle.Render(
						fmt.Sprintf(
							"Rows %d–%d of %d",
							start+1,
							end,
							len(m.dynoPoints),
						),
					),
				),
			)
	}

	// Controls shown directly below the data table.
	lines =
		append(
			lines,
			lineStyle.Render(""),
			lineStyle.Render(
				mutedStyle.Render(
					"[↑↓] Navigate  [TAB] Next  [SHIFT+TAB] Previous",
				),
			),
			lineStyle.Render(
				mutedStyle.Render(
					"[M] Switch graph  [ENTER/E] Edit  [G] Fullscreen  [C] Customize  [ESC] Back",
				),
			),
			lineStyle.Render(
				mutedStyle.Render(
					"[E] Export PNG",
				),
			),
		)

	return strings.Join(
		lines,
		"\n",
	)
}

// ─────────────────────────────────────────────────────────────
// Graph
// ─────────────────────────────────────────────────────────────

func dynoGraphView(
	p palette,
	width int,
	height int,
) string {

	return dynoGraphText(
		nil,
		width,
		height,
		p,
	)
}

func dynoModeSwitchText(mode dynoGraphMode, p palette) string {
	active := lipgloss.NewStyle().Bold(true).Foreground(p.accent).Underline(true)
	inactive := lipgloss.NewStyle().Foreground(p.muted)
	first, second := "[BHP vs RPM]", "[Torque vs SPS]"
	if mode == dynoGraphTorque {
		return lipgloss.JoinHorizontal(lipgloss.Left, inactive.Render(first), " ", active.Render(second))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, active.Render(first), " ", inactive.Render(second))
}

func dynoGraphText(
	points []dynoGraphPoint,
	width int,
	height int,
	p palette,
	modes ...dynoGraphMode,
) string {

	titleStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	mutedStyle :=
		lipgloss.NewStyle().
			Foreground(p.muted)

	axisStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.text)

	curveStyle :=
		lipgloss.NewStyle().
			Foreground(p.accent)

	axisTickStyle :=
		lipgloss.NewStyle().
			Foreground(p.muted).
			Background(p.panel)

	mode := dynoGraphBHP
	if len(modes) > 0 {
		mode = modes[0]
	}
	graphPoints := dynoGraphSeries(points, mode)

	if len(graphPoints) == 0 || (len(graphPoints) == 1 && graphPoints[0].RPM == 0 && graphPoints[0].BHP == 0) {

		return strings.Join(
			[]string{
				titleStyle.Render(
					dynoGraphModeTitle(mode),
				),
				"",
				mutedStyle.Render(
					"Enter SPS and torque data.",
				),
			},
			"\n",
		)
	}

	if width < 24 {
		width = 24
	}

	if height < 8 {
		height = 8
	}

	const yLabelWidth = 8

	graphWidth :=
		width -
			yLabelWidth - 4

	if graphWidth < 12 {
		graphWidth = 12
	}

	// Keep the graph tall. Only a small amount of vertical
	// space is reserved for the RPM scale underneath it.
	graphHeight :=
		height - 4

	if graphHeight < 6 {
		graphHeight = 6
	}

	minRPM,
		maxRPM,
		minBHP,
		maxBHP :=
		dynoGraphBounds(graphPoints)

	pixelWidth :=
		graphWidth * 2

	pixelHeight :=
		graphHeight * 4

	if pixelWidth < 2 {
		pixelWidth = 2
	}

	if pixelHeight < 4 {
		pixelHeight = 4
	}

	pixels :=
		make(
			[][]bool,
			pixelHeight,
		)

	for y := range pixels {

		pixels[y] =
			make(
				[]bool,
				pixelWidth,
			)
	}

	scaleX :=
		func(rpm float64) int {

			if maxRPM <= minRPM {
				return 0
			}

			ratio :=
				(rpm - minRPM) /
					(maxRPM - minRPM)

			x :=
				int(
					math.Round(
						ratio *
							float64(
								pixelWidth-1,
							),
					),
				)

			if x < 0 {
				x = 0
			}

			if x >= pixelWidth {
				x =
					pixelWidth -
						1
			}

			return x
		}

	scaleY :=
		func(bhp float64) int {

			if maxBHP <= minBHP {
				return 0
			}

			ratio :=
				(bhp - minBHP) /
					(maxBHP - minBHP)

			y :=
				int(
					math.Round(
						(1 - ratio) *
							float64(
								pixelHeight-1,
							),
					),
				)

			if y < 0 {
				y = 0
			}

			if y >= pixelHeight {
				y =
					pixelHeight -
						1
			}

			return y
		}

	// Shape-preserving interpolation prevents the overshoot and
	// loops that Catmull-Rom can introduce.

	if len(graphPoints) == 1 {

		x := scaleX(graphPoints[0].RPM)
		y := scaleY(graphPoints[0].BHP)

		dynoDrawPixelLine(
			pixels,
			x,
			y,
			x,
			y,
		)

	} else if len(graphPoints) == 2 {

		dynoDrawPixelLine(
			pixels,
			scaleX(graphPoints[0].RPM),
			scaleY(graphPoints[0].BHP),
			scaleX(graphPoints[1].RPM),
			scaleY(graphPoints[1].BHP),
		)

	} else {

		slopes :=
			dynoPCHIPSlopes(
				graphPoints,
			)

		if slopes == nil {

			for i := 0; i < len(graphPoints)-1; i++ {

				dynoDrawPixelLine(
					pixels,
					scaleX(graphPoints[i].RPM),
					scaleY(graphPoints[i].BHP),
					scaleX(graphPoints[i+1].RPM),
					scaleY(graphPoints[i+1].BHP),
				)
			}

		} else {

			const samplesPerSegment = 24

			for i := 0; i < len(graphPoints)-1; i++ {

				p1 := graphPoints[i]
				p2 := graphPoints[i+1]

				xPrevious := scaleX(p1.RPM)
				yPrevious := scaleY(p1.BHP)

				for sample := 1; sample <= samplesPerSegment; sample++ {

					t :=
						float64(sample) /
							float64(samplesPerSegment)

					rpm :=
						p1.RPM +
							(p2.RPM-p1.RPM)*t

					bhp :=
						dynoPCHIPValue(
							p1,
							p2,
							slopes[i],
							slopes[i+1],
							t,
						)

					x := scaleX(rpm)
					y := scaleY(bhp)

					dynoDrawPixelLine(
						pixels,
						xPrevious,
						yPrevious,
						x,
						y,
					)

					xPrevious = x
					yPrevious = y
				}
			}
		}
	}

	var lines []string

	lines =
		append(
			lines,
			titleStyle.Render(
				dynoGraphModeTitle(mode),
			),
			axisStyle.Render(
				strings.Repeat(
					" ",
					yLabelWidth,
				)+
					dynoGraphYAxisLabel(mode)+" ↑",
			),
		)

	// ────────────────────────────────────────────────────────
	// Plot
	// ────────────────────────────────────────────────────────

	for row := 0; row < graphHeight; row++ {

		// BHP labels use pixel-space, just like the actual graph.
		pixelY :=
			row * 4

		ratio :=
			float64(pixelY) /
				float64(
					pixelHeight-1,
				)

		bhpValue :=
			maxBHP -
				(maxBHP-minBHP)*
					ratio

		label :=
			fmt.Sprintf(
				"%7.1f ",
				bhpValue,
			)

		var graph strings.Builder

		for col := 0; col < graphWidth; col++ {

			var bits int

			for py := 0; py < 4; py++ {
				for px := 0; px < 2; px++ {
					x := col*2 + px
					y := row*4 + py
					if x < 0 || y < 0 || x >= pixelWidth || y >= pixelHeight {
						continue
					}
					if !pixels[y][x] {
						continue
					}
					bits |= int(dynoBrailleBit(px, py))
				}
			}

			if bits == 0 {
				graph.WriteByte(' ')
				continue
			}
			graph.WriteString(curveStyle.Render(string(rune(0x2800 + bits))))
		}

		leftTick, rightTick := "─┤", "├─"

		lines =
			append(
				lines,
				mutedStyle.Render(label)+
					axisTickStyle.Render(leftTick)+
					graph.String()+
					axisTickStyle.Render(rightTick),
			)
	}

	// ────────────────────────────────────────────────────────
	// X axis
	// ────────────────────────────────────────────────────────

	graphPrefix :=
		strings.Repeat(
			" ",
			yLabelWidth+2,
		)

	// Redundant X-axis baseline omitted; short ticks and numeric labels remain.

	// ────────────────────────────────────────────────────────
	// RPM numbers
	// ────────────────────────────────────────────────────────
	//
	// scaleX() gives the exact Braille-pixel X coordinate.
	// Divide by 2 because each terminal character is two
	// horizontal Braille pixels wide.
	// ────────────────────────────────────────────────────────

	tickCount :=
		graphWidth / 5

	if tickCount < 5 {
		tickCount = 5
	}

	if tickCount > 14 {
		tickCount = 14
	}

	rpmLine :=
		make(
			[]rune,
			graphWidth,
		)

	for i := range rpmLine {
		rpmLine[i] = ' '
	}

	type rpmLabel struct {
		start int
		end   int
		text  string
	}

	labels :=
		make(
			[]rpmLabel,
			0,
			tickCount,
		)

	for i := 0; i < tickCount; i++ {

		ratio :=
			float64(i) /
				float64(
					tickCount-1,
				)

		rpm :=
			minRPM +
				(maxRPM-minRPM)*
					ratio

		pixelX :=
			scaleX(rpm)

		cellX :=
			pixelX / 2

		text :=
			fmt.Sprintf(
				"%.0f",
				rpm,
			)

		textWidth :=
			len(text)

		start :=
			cellX -
				textWidth/2

		if start < 0 {
			start = 0
		}

		if start+textWidth >
			graphWidth {

			start =
				graphWidth -
					textWidth
		}

		if start < 0 {
			continue
		}

		labels =
			append(
				labels,
				rpmLabel{
					start: start,
					end: start +
						textWidth -
						1,
					text: text,
				},
			)
	}

	// Remove overlapping labels from the inside while
	// preserving the first and last labels.
	for len(labels) > 1 {

		overlapIndex := -1

		for i := 1; i < len(labels); i++ {

			if labels[i].start <=
				labels[i-1].end {

				overlapIndex = i
				break
			}
		}

		if overlapIndex == -1 {
			break
		}

		if overlapIndex == len(labels)-1 {

			labels =
				append(
					labels[:overlapIndex],
					labels[overlapIndex+1:]...,
				)

			continue
		}

		labels =
			append(
				labels[:overlapIndex],
				labels[overlapIndex+1:]...,
			)
	}

	// Place X-axis tick marks outside the plotting area, aligned with
	// the numeric labels. The curve area itself stays free of grid lines.
	xTickLine := make([]rune, graphWidth)
	for i := range xTickLine {
		xTickLine[i] = ' '
	}
	for _, label := range labels {
		center := label.start + len(label.text)/2
		if center >= 0 && center < len(xTickLine) {
			xTickLine[center] = '╷'
		}
	}
	lines = append(
		lines,
		mutedStyle.Render(graphPrefix+string(xTickLine)),
	)

	for _, label := range labels {

		for i, r := range []rune(label.text) {

			pos :=
				label.start + i

			if pos >= 0 &&
				pos < graphWidth {

				rpmLine[pos] = r
			}
		}
	}

	lines =
		append(
			lines,
			mutedStyle.Render(
				graphPrefix+
					string(rpmLine),
			),
		)

	rpmAxisPadding :=
		graphWidth/2 - 2

	if rpmAxisPadding < 0 {
		rpmAxisPadding = 0
	}

	lines =
		append(
			lines,
			axisStyle.Render(
				graphPrefix+
					strings.Repeat(
						" ",
						rpmAxisPadding,
					)+
					dynoGraphXAxisLabel(mode)+" →",
			),
		)

	return strings.Join(
		lines,
		"\n",
	)
}

func dynoBrailleBit(
	x int,
	y int,
) byte {

	switch {

	case x == 0 && y == 0:
		return 1

	case x == 0 && y == 1:
		return 2

	case x == 0 && y == 2:
		return 4

	case x == 1 && y == 0:
		return 8

	case x == 1 && y == 1:
		return 16

	case x == 1 && y == 2:
		return 32

	case x == 0 && y == 3:
		return 64

	case x == 1 && y == 3:
		return 128

	default:
		return 0
	}
}

func dynoDrawPixelLine(
	pixels [][]bool,
	x0 int,
	y0 int,
	x1 int,
	y1 int,
) {

	dx :=
		math.Abs(
			float64(
				x1 - x0,
			),
		)

	dy :=
		math.Abs(
			float64(
				y1 - y0,
			),
		)

	steps :=
		int(
			math.Max(
				dx,
				dy,
			),
		)

	if steps == 0 {

		if y0 >= 0 &&
			y0 < len(pixels) &&
			x0 >= 0 &&
			x0 < len(pixels[y0]) {

			pixels[y0][x0] =
				true
		}

		return
	}

	for i := 0; i <= steps; i++ {

		t :=
			float64(i) /
				float64(steps)

		x :=
			int(
				math.Round(
					float64(x0) +
						float64(x1-x0)*t,
				),
			)

		y :=
			int(
				math.Round(
					float64(y0) +
						float64(y1-y0)*t,
				),
			)

		if y >= 0 &&
			y < len(pixels) &&
			x >= 0 &&
			x < len(pixels[y]) {

			pixels[y][x] =
				true
		}
	}
}

// ─────────────────────────────────────────────────────────────
// Results
// ─────────────────────────────────────────────────────────────

func (m model) dynoResultsView(
	p palette,
) string {

	title :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	header :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.text)

	muted :=
		lipgloss.NewStyle().
			Foreground(p.muted)

	points :=
		m.dynoGraphPoints()

	lines :=
		[]string{
			title.Render(
				"DYNO RESULTS",
			),
			"",
			header.Render(
				"#    SPS        RPM       TORQUE        BHP",
			),
			muted.Render(
				"────────────────────────────────────────────",
			),
		}

	measurements := dynoMeasuredPoints(points)

	for i, point := range measurements {

		lines =
			append(
				lines,
				fmt.Sprintf(
					"%-3d  %8.3f   %8.3f   %8.3f   %8.3f",
					i+1,
					point.SPS,
					point.RPM,
					point.Torque,
					point.BHP,
				),
			)
	}

	if len(measurements) == 0 {

		lines =
			append(
				lines,
				"",
				muted.Render(
					"No complete rows.",
				),
			)

	}

	return strings.Join(
		lines,
		"\n",
	)
}

func (m model) dynoPeaksView(
	p palette,
) string {

	muted :=
		lipgloss.NewStyle().
			Foreground(p.muted)

	points :=
		m.dynoGraphPoints()

	lines :=
		[]string{
			muted.Render(
				"PEAKS",
			),
		}

	if len(dynoMeasuredPoints(points)) == 0 {

		lines =
			append(
				lines,
				"No complete rows.",
			)

		return strings.Join(
			lines,
			"\n",
		)
	}

	peakTorque,
		peakTorqueSPS,
		_ :=
		dynoPeakTorque(points)

	peakSPS,
		peakSPSTorque,
		_ :=
		dynoPeakSPS(points)

	peakBHP,
		peakBHPSPS,
		_ :=
		dynoPeakBHP(points)

	lines =
		append(
			lines,
			fmt.Sprintf(
				"Peak Torque : %.3f @ %.3f SPS",
				peakTorque,
				peakTorqueSPS,
			),
			fmt.Sprintf(
				"Peak SPS    : %.3f @ %.3f Torque",
				peakSPS,
				peakSPSTorque,
			),
			fmt.Sprintf(
				"Peak BHP    : %.3f @ %.3f SPS",
				peakBHP,
				peakBHPSPS,
			),
		)

	return strings.Join(
		lines,
		"\n",
	)
}

// ─────────────────────────────────────────────────────────────
// PNG export
// ─────────────────────────────────────────────────────────────

func dynoExportPNG(m model) {
	points := m.dynoGraphPoints()
	graphPoints := dynoGraphSeries(points, m.dynoMode)
	// dynoGraphPoints always includes the origin; without a measured row
	// the TUI shows its empty state, so do not export a misleading empty plot.
	if len(points) <= 1 || len(graphPoints) <= 1 {
		return
	}

	const (
		imgWidth   = 1600
		imgHeight  = 900
		plotLeft   = 175
		plotRight  = 1490
		plotTop    = 125
		plotBottom = 730
	)

	p := getPalette(m.theme, m.accent)
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	for y := 0; y < imgHeight; y++ {
		for x := 0; x < imgWidth; x++ {
			img.Set(x, y, p.panel)
		}
	}

	minRPM, maxRPM, minBHP, maxBHP := dynoGraphBounds(graphPoints)
	scaleX := func(value float64) int {
		return plotLeft + dynoScaleX(value, minRPM, maxRPM, plotRight-plotLeft+1)
	}
	scaleY := func(value float64) int {
		return plotTop + dynoScaleY(value, minBHP, maxBHP, plotBottom-plotTop+1)
	}

	// Keep the plot itself clean: only the axes, with short outward ticks.
	dynoPNGLine(img, plotLeft, plotTop, plotLeft, plotBottom, p.border)
	dynoPNGLine(img, plotRight, plotTop, plotRight, plotBottom, p.border)
	dynoPNGLine(img, plotLeft, plotBottom, plotRight, plotBottom, p.border)

	const tickDivisions = 5
	for i := 0; i <= tickDivisions; i++ {
		ratio := float64(i) / float64(tickDivisions)
		y := plotTop + int(math.Round(ratio*float64(plotBottom-plotTop)))
		value := maxBHP - (maxBHP-minBHP)*ratio
		// Short horizontal Y-axis ticks outside both plot edges.
		dynoPNGLine(img, plotLeft-8, y, plotLeft, y, p.muted)
		dynoPNGLine(img, plotRight, y, plotRight+8, y, p.muted)
		label := fmt.Sprintf("%.1f", value)
		dynoPNGText(img, plotLeft-16-dynoPNGTextWidth(label, 2), y-7, 2, label, p.muted)

		x := plotLeft + int(math.Round(ratio*float64(plotRight-plotLeft)))
		xValue := minRPM + (maxRPM-minRPM)*ratio
		// Short vertical X-axis ticks below the baseline; no second bar.
		dynoPNGLine(img, x, plotBottom, x, plotBottom+8, p.muted)
		xLabel := fmt.Sprintf("%.0f", xValue)
		dynoPNGText(img, x-dynoPNGTextWidth(xLabel, 2)/2, plotBottom+17, 2, xLabel, p.muted)
	}

	title := strings.ToUpper(dynoGraphModeTitle(m.dynoMode))
	yLabel := strings.ToUpper(dynoGraphYAxisLabel(m.dynoMode))
	xLabel := strings.ToUpper(dynoGraphXAxisLabel(m.dynoMode))
	dynoPNGText(img, plotLeft, 27, 3, title, p.text)
	dynoPNGText(img, plotLeft, 77, 2, yLabel, p.muted)
	dynoPNGText(img, plotLeft+(plotRight-plotLeft-dynoPNGTextWidth(xLabel, 2))/2, plotBottom+57, 2, xLabel, p.muted)

	// Use shape-preserving PCHIP, sampled for the PNG's actual pixel resolution.
	slopes := dynoPCHIPSlopes(graphPoints)
	if slopes == nil {
		for i := 1; i < len(graphPoints); i++ {
			dynoPNGThickLine(img,
				scaleX(graphPoints[i-1].RPM), scaleY(graphPoints[i-1].BHP),
				scaleX(graphPoints[i].RPM), scaleY(graphPoints[i].BHP), p.accent)
		}
	} else {
		for i := 0; i < len(graphPoints)-1; i++ {
			p1, p2 := graphPoints[i], graphPoints[i+1]
			xPrevious, yPrevious := scaleX(p1.RPM), scaleY(p1.BHP)
			xEnd, yEnd := scaleX(p2.RPM), scaleY(p2.BHP)
			pixelSpan := absInt(xEnd - xPrevious)
			if verticalSpan := absInt(yEnd - yPrevious); verticalSpan > pixelSpan {
				pixelSpan = verticalSpan
			}
			// Aim for at least ~1.5 samples per pixel of travel. Keep a
			// floor for short segments and a cap for pathological inputs.
			samplesPerSegment := int(math.Ceil(float64(pixelSpan) * 1.5))
			if samplesPerSegment < 24 {
				samplesPerSegment = 24
			}
			if samplesPerSegment > 8192 {
				samplesPerSegment = 8192
			}
			for sample := 1; sample <= samplesPerSegment; sample++ {
				t := float64(sample) / float64(samplesPerSegment)
				rpm := p1.RPM + (p2.RPM-p1.RPM)*t
				bhp := dynoPCHIPValue(p1, p2, slopes[i], slopes[i+1], t)
				x, y := scaleX(rpm), scaleY(bhp)
				dynoPNGThickLine(img, xPrevious, yPrevious, x, y, p.accent)
				xPrevious, yPrevious = x, y
			}
		}
	}

	exportName := "dyno.png"
	if m.dynoMode == dynoGraphTorque {
		exportName = "dyno-torque-sps.png"
	}
	exportPath := exportName
	if m.cfg.ExportDirectory != "" {
		if err := os.MkdirAll(m.cfg.ExportDirectory, 0755); err != nil {
			return
		}
		exportPath = filepath.Join(m.cfg.ExportDirectory, exportName)
	}
	file, err := os.Create(exportPath)
	if err != nil {
		return
	}
	defer file.Close()
	_ = png.Encode(file, img)
}

func dynoPNGThickLine(img *image.RGBA, x0, y0, x1, y1 int, col color.Color) {
	// Three-pixel stroke, offset perpendicular to the dominant direction.
	if absInt(x1-x0) >= absInt(y1-y0) {
		for offset := -1; offset <= 1; offset++ {
			dynoPNGLine(img, x0, y0+offset, x1, y1+offset, col)
		}
		return
	}
	for offset := -1; offset <= 1; offset++ {
		dynoPNGLine(img, x0+offset, y0, x1+offset, y1, col)
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func dynoPNGTextWidth(text string, scale int) int {
	if len([]rune(text)) == 0 {
		return 0
	}
	return (len([]rune(text))*6 - 1) * scale
}

func dynoPNGText(img *image.RGBA, x, y, scale int, text string, col color.Color) {
	if scale < 1 {
		return
	}
	cursorX := x
	for _, ch := range strings.ToUpper(text) {
		glyph, ok := dynoPNGGlyphs[ch]
		if !ok {
			glyph = dynoPNGGlyphs[' ']
		}
		for row, pattern := range glyph {
			for column, bit := range pattern {
				if bit != '1' {
					continue
				}
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						px := cursorX + column*scale + sx
						py := y + row*scale + sy
						if image.Pt(px, py).In(img.Bounds()) {
							img.Set(px, py, col)
						}
					}
				}
			}
		}
		cursorX += 6 * scale
	}
}

var dynoPNGGlyphs = map[rune][7]string{
	' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01111", "10000", "10000", "10111", "10001", "10001", "01111"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'J': {"00111", "00010", "00010", "00010", "10010", "10010", "01100"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'.': {"00000", "00000", "00000", "00000", "00000", "00110", "00110"},
	'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
}

func dynoScaleX(
	value float64,
	minValue float64,
	maxValue float64,
	pixelWidth int,
) int {

	if pixelWidth <= 1 ||
		maxValue <= minValue {

		return 0
	}

	ratio :=
		(value - minValue) /
			(maxValue - minValue)

	x :=
		int(
			math.Round(
				ratio *
					float64(
						pixelWidth-1,
					),
			),
		)

	if x < 0 {
		x = 0
	}

	if x >= pixelWidth {
		x =
			pixelWidth -
				1
	}

	return x
}

func dynoScaleY(
	value float64,
	minValue float64,
	maxValue float64,
	pixelHeight int,
) int {

	if pixelHeight <= 1 ||
		maxValue <= minValue {

		return 0
	}

	ratio :=
		(value - minValue) /
			(maxValue - minValue)

	y :=
		int(
			math.Round(
				(1 - ratio) *
					float64(
						pixelHeight-1,
					),
			),
		)

	if y < 0 {
		y = 0
	}

	if y >= pixelHeight {
		y =
			pixelHeight -
				1
	}

	return y
}

func dynoPNGLine(
	img *image.RGBA,
	x0 int,
	y0 int,
	x1 int,
	y1 int,
	col color.Color,
) {

	dx :=
		math.Abs(
			float64(
				x1 - x0,
			),
		)

	dy :=
		math.Abs(
			float64(
				y1 - y0,
			),
		)

	steps :=
		int(
			math.Max(
				dx,
				dy,
			),
		)

	if steps == 0 {

		img.Set(
			x0,
			y0,
			col,
		)

		return
	}

	for i := 0; i <= steps; i++ {

		t :=
			float64(i) /
				float64(steps)

		x :=
			int(
				math.Round(
					float64(x0) +
						float64(x1-x0)*t,
				),
			)

		y :=
			int(
				math.Round(
					float64(y0) +
						float64(y1-y0)*t,
				),
			)

		if x >= 0 &&
			x < img.Bounds().Dx() &&
			y >= 0 &&
			y < img.Bounds().Dy() {

			img.Set(
				x,
				y,
				col,
			)
		}
	}
}

func dynoPNGCircle(
	img *image.RGBA,
	cx int,
	cy int,
	radius int,
	col color.Color,
) {

	for y := -radius; y <= radius; y++ {

		for x := -radius; x <= radius; x++ {

			if x*x+y*y >
				radius*radius {

				continue
			}

			px :=
				cx + x

			py :=
				cy + y

			if px >= 0 &&
				px < img.Bounds().Dx() &&
				py >= 0 &&
				py < img.Bounds().Dy() {

				img.Set(
					px,
					py,
					col,
				)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────
// Normal Dyno view
// ─────────────────────────────────────────────────────────────

func (m model) viewDyno() tea.View {

	if m.dynoFullscreen {
		return m.viewDynoFullscreen()
	}

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

	panelHeight :=
		height - 2

	if panelHeight < 1 {
		panelHeight = 1
	}

	leftWidth :=
		width * 32 / 100

	if leftWidth < 36 {
		leftWidth = 36
	}

	if leftWidth > width-20 {
		leftWidth =
			width - 20
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

	leftStyle :=
		lipgloss.NewStyle().
			Width(leftWidth).
			Height(panelHeight).
			Padding(1, 2).
			Foreground(p.text)

	rightStyle :=
		lipgloss.NewStyle().
			Width(rightWidth).
			Height(panelHeight).
			Padding(1, 2).
			Foreground(p.text)

	if m.cfg.Appearance.Background !=
		"transparent" {

		leftStyle =
			leftStyle.Background(
				p.panel,
			)

		rightStyle =
			rightStyle.Background(
				p.panel,
			)
	}

	leftContent :=
		strings.Join(
			[]string{
				m.dynoDataView(p),
				"",
				m.dynoPeaksView(p),
			},
			"\n",
		)

	leftPanel :=
		leftStyle.Render(
			leftContent,
		)

	graphHeight :=
		panelHeight/2 - 1

	if graphHeight < 9 {
		graphHeight = 9
	}

	if graphHeight > panelHeight {
		graphHeight = panelHeight
	}

	modeSwitch := dynoModeSwitchText(m.dynoMode, p)
	graph :=
		dynoGraphText(
			m.dynoGraphPoints(),
			rightWidth-4,
			graphHeight,
			p,
			m.dynoMode,
		)

	results :=
		m.dynoResultsView(
			p,
		)

	rightContent :=
		strings.Join(
			[]string{
				modeSwitch,
				graph,
				"",
				results,
			},
			"\n",
		)

	rightPanel :=
		rightStyle.Render(
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

	if m.cfg.Dyno.GraphSide == "left" {
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

	topStyle :=
		lipgloss.NewStyle().
			Width(width).
			Height(2).
			Bold(true).
			Foreground(p.accent)

	if m.cfg.Appearance.Background !=
		"transparent" {

		topStyle =
			topStyle.Background(
				p.surface,
			)
	}

	dynoTitleText := "PC MULTITOOL  •  DYNO"
	dynoAuthorText := "Made by Xad0"

	dynoTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(p.accent)

	dynoAuthorStyle := lipgloss.NewStyle().
		Foreground(p.muted)

	if m.cfg.Appearance.Background != "transparent" {
		dynoTitleStyle =
			dynoTitleStyle.Background(p.surface)
		dynoAuthorStyle =
			dynoAuthorStyle.Background(p.surface)
	}

	dynoTitle := dynoTitleStyle.Render(dynoTitleText)
	dynoAuthor := dynoAuthorStyle.Render(dynoAuthorText)

	dynoGap := width -
		lipgloss.Width(dynoTitleText) -
		lipgloss.Width(dynoAuthorText)

	if dynoGap < 1 {
		dynoGap = 1
	}

	dynoTopStyle :=
		lipgloss.NewStyle().
			Width(width).
			Height(2).
			Bold(true)

	if m.cfg.Appearance.Background != "transparent" {
		dynoTopStyle =
			dynoTopStyle.Background(p.surface)
	}

	dynoTop :=
		dynoTopStyle.Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				dynoTitle,
				strings.Repeat(" ", dynoGap),
				dynoAuthor,
			),
		)

	content :=
		strings.Join(
			[]string{
				dynoTop,
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

// ─────────────────────────────────────────────────────────────
// Fullscreen
// ─────────────────────────────────────────────────────────────

func (m model) viewDynoFullscreen() tea.View {

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

	title :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent).
			Render(
				"DYNO • " + dynoGraphModeTitle(m.dynoMode),
			)

	modeSwitch := dynoModeSwitchText(m.dynoMode, p)
	graph :=
		dynoGraphText(
			m.dynoGraphPoints(),
			width-6,
			height-9,
			p,
			m.dynoMode,
		)

	footer :=
		lipgloss.NewStyle().
			Foreground(p.muted).
			Render(
				"[G / ESC] Back    [M] Graph mode    [E] Export PNG",
			)

	content :=
		strings.Join(
			[]string{
				title,
				"",
				modeSwitch,
				"",
				graph,
				"",
				footer,
			},
			"\n",
		)

	style :=
		lipgloss.NewStyle().
			Width(width).
			Height(height).
			Padding(1, 2).
			Foreground(p.text)

	if m.cfg.Appearance.Background !=
		"transparent" {

		style =
			style.Background(
				p.panel,
			)
	}

	view :=
		tea.NewView(
			style.Render(
				content,
			),
		)

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
