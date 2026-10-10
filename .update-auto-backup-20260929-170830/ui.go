package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type page int

const (
	pageHome page = iota
	pageSetup
	pageMain
	pageCrank
	pageWheel
	pageDyno
	pagePiston
	pageSettings
	pageUpdates
	pageCustomize
	pageToolCustomizeSelect
	pageAppearance
	pageThemes
	pageFlavors
	pageAccents
	pageTransparency
	pageLayout
	pageReference
	pageCalculator
	pageReset
	pageAbout
)

type customizeTool int

const (
	customizeGear customizeTool = iota
	customizeCrank
	customizeWheel
	customizeDyno
	customizePiston
)

func customizeToolName(tool customizeTool) string {
	switch tool {
	case customizeGear:
		return "Gear Calculator"
	case customizeCrank:
		return "Crank Angle Calculator"
	case customizeWheel:
		return "Wheel Calculator"
	case customizeDyno:
		return "Dyno"
	case customizePiston:
		return "Piston Length Calculator"
	default:
		return "Unknown"
	}
}

func customizeToolPage(tool customizeTool) page {
	switch tool {
	case customizeGear:
		return pageMain
	case customizeCrank:
		return pageCrank
	case customizeWheel:
		return pageWheel
	case customizeDyno:
		return pageDyno
	case customizePiston:
		return pagePiston
	default:
		return pageHome
	}
}

type model struct {
	teethInput      textinput.Model
	compressorInput textinput.Model

	page   page
	width  int
	height int

	cfg config

	theme  string
	accent string

	customizeTool   customizeTool
	customToolIndex int

	homeIndex         int
	themeIndex        int
	accentIndex       int
	customIndex       int
	appearanceIndex   int
	transparencyIndex int
	layoutIndex       int
	referenceIndex    int
	calculatorIndex   int
	resetIndex        int
	settingsIndex     int
	updatesIndex      int

	updateInfo      *updateInfo
	updateStatus    string
	updateChecking  bool
	updateReadyPath string

	referenceSmallestInput textinput.Model
	referenceMaximumInput  textinput.Model
	referenceScroll        int

	crankIndex            int
	crankLayoutIndex      int
	crankCylinderInput    textinput.Model
	catppuccinFlavorIndex int

	wheelIndex     int
	wheelTireInput textinput.Model

	pistonDistanceInput textinput.Model
	pistonAmountInput   textinput.Model
	pistonValueInput    textinput.Model
	pistonFieldIndex    int

	dynoPoints     []dynoPoint
	dynoFieldIndex int
	dynoScroll     int
	dynoFullscreen bool

	setupIndex       int
	setupExportInput textinput.Model
}

func newInput(value string) textinput.Model {
	input := textinput.New()
	input.SetValue(value)
	input.SetWidth(14)
	input.CharLimit = 8
	input.Prompt = ""
	input.Blur()
	return input
}

func initialModel() model {
	cfg := loadConfig()

	teeth := 6
	_, _, offset := Calculate(teeth)
	compressors := AutoCompressors(offset)

	setupInput := textinput.New()
	setupInput.SetWidth(60)
	setupInput.CharLimit = 256
	setupInput.Prompt = ""
	setupInput.Blur()

	referenceSmallestInput :=
		newInput(
			strconv.Itoa(
				cfg.Reference.SmallestGear,
			),
		)

	referenceMaximumInput :=
		newInput(
			strconv.Itoa(
				cfg.Reference.MaximumGear,
			),
		)

	m := model{
		teethInput:             newInput(strconv.Itoa(teeth)),
		compressorInput:        newInput(strconv.Itoa(compressors)),
		referenceSmallestInput: referenceSmallestInput,
		referenceMaximumInput:  referenceMaximumInput,
		referenceScroll:        0,

		crankCylinderInput:  crankInput(6),
		wheelTireInput:      wheelInput(10),
		pistonDistanceInput: newInput("1"),
		pistonAmountInput:   newInput("2"),
		pistonValueInput:    newInput("3.106"),
		dynoPoints:          initialDynoPoints(),
		setupExportInput:    setupInput,

		page:   pageHome,
		width:  100,
		height: 24,

		cfg:    cfg,
		theme:  cfg.Appearance.Theme,
		accent: cfg.Appearance.Accent,

		customizeTool:   customizeGear,
		customToolIndex: 0,

		homeIndex:        0,
		crankLayoutIndex: int(crankInline),
		dynoFieldIndex:   0,
		dynoScroll:       0,
		dynoFullscreen:   false,
		setupIndex:       0,
		settingsIndex:    0,
		updatesIndex:     0,
	}

	m.updateThemeIndex()
	m.updateAccentIndex()

	if !cfg.SetupCompleted {
		m.page = pageSetup
	}

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case updateCheckMsg:
		m.updateChecking = false

		if msg.err != nil {
			m.updateInfo = nil
			m.updateStatus = "Unable to check for updates."
			return m, nil
		}

		m.updateInfo = &msg.info

		if isNewerVersion(
			msg.info.LatestVersion,
			msg.info.CurrentVersion,
		) {
			m.updateStatus = "Update available."
		} else {
			m.updateStatus = "You are up to date."
		}

		return m, nil

	case updateDownloadMsg:
		m.updateChecking = false

		if msg.err != nil {
			m.updateStatus =
				"Update failed: " + msg.err.Error()
			return m, nil
		}

		if msg.path == "" {
			m.updateStatus =
				"Update failed: no downloaded application."
			return m, nil
		}

		if err :=
			startUpdateHelper(
				msg.path,
			); err != nil {

			os.Remove(msg.path)

			m.updateStatus =
				"Unable to start updater: " +
					err.Error()

			return m, nil
		}

		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.referenceScroll =
			m.clampReferenceScroll()

		return m, nil

	case tea.MouseClickMsg:
		return m.handleMouse(msg)

	case tea.MouseWheelMsg:

		if m.page == pageDyno {
			return m.handleDynoWheel(msg)
		}

		if m.page == pageMain &&
			m.cfg.Reference.Enabled &&
			m.referenceMouseArea(
				msg.X,
				msg.Y,
			) {

			switch msg.Button {

			case tea.MouseWheelDown:
				m.referenceScroll++

			case tea.MouseWheelUp:
				m.referenceScroll--
			}

			m.referenceScroll =
				m.clampReferenceScroll()

			return m, nil
		}

		return m, nil

	case tea.KeyPressMsg:

		switch m.page {

		case pageHome:
			return m.updateHome(msg)

		case pageSetup:
			return m.updateSetup(msg)

		case pageMain:
			return m.updateMain(msg)

		case pageCrank:
			return m.updateCrank(msg)

		case pageWheel:
			return m.updateWheel(msg)

		case pageDyno:
			return m.updateDyno(msg)

		case pagePiston:
			return m.updatePiston(msg)

		case pageSettings:
			return m.updateSettings(msg)

		case pageUpdates:
			return m.updateUpdates(msg)

		case pageCustomize:
			return m.updateCustomize(msg)

		case pageToolCustomizeSelect:
			return m.updateToolCustomizeSelect(msg)

		case pageAppearance:
			return m.updateAppearance(msg)

		case pageThemes:
			return m.updateThemes(msg)

		case pageFlavors:
			return m.updateCatppuccinFlavors(msg)

		case pageAccents:
			return m.updateAccents(msg)

		case pageTransparency:
			return m.updateTransparency(msg)

		case pageLayout:
			return m.updateLayout(msg)

		case pageReference:
			return m.updateReference(msg)

		case pageCalculator:
			return m.updateCalculator(msg)

		case pageReset:
			return m.updateReset(msg)

		case pageAbout:

			switch msg.String() {

			case "q", "ctrl+c":
				return m, tea.Quit

			case "esc":
				m.page = pageHome
				m.homeIndex = 5
			}

			return m, nil
		}
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// PC Multitool launcher
// ─────────────────────────────────────────────────────────────

func (m model) updateHome(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := []string{
		"Gear Calculator",
		"Crank Angle Calculator",
		"Wheel Calculator",
		"Dyno",
		"Piston Length Calculator",
		"Settings",
		"Updates",
		"About",
		"Quit",
	}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		next := m.homeIndex - 1
		if next < 0 {
			next = len(items) - 1
		}

		m.homeIndex = next

	case "down":

		next := m.homeIndex + 1
		if next >= len(items) {
			next = 0
		}

		m.homeIndex = next

	case "enter":
		return m.openHomeItem(m.homeIndex)
	}

	return m, nil
}

func (m model) openHomeItem(index int) (tea.Model, tea.Cmd) {
	m.homeIndex = index

	switch index {

	case 0:
		m.stopEditing()
		m.referenceScroll = 0
		m.page = pageMain

	case 1:
		m.stopEditing()
		m.crankLayoutIndex = int(crankInline)
		m.crankCylinderInput.Blur()
		m.page = pageCrank

	case 2:
		m.stopEditing()
		m.wheelTireInput.Blur()
		m.page = pageWheel

	case 3:
		m.stopEditing()
		m.page = pageDyno
		return m, m.dynoFocusField(0)

	case 4:
		m.stopEditing()
		m.pistonFieldIndex = 0
		m.page = pagePiston

	case 5:
		m.stopEditing()
		m.customIndex = 0
		m.page = pageSettings

	case 6:
		m.stopEditing()
		m.updateInfo = nil
		m.updateStatus = ""
		m.updateChecking = false
		m.updatesIndex = 0
		m.page = pageUpdates

	case 7:
		m.stopEditing()
		m.page = pageAbout

	case 8:
		return m, tea.Quit
	}

	return m, nil
}

func (m model) viewHome() tea.View {

	p :=
		getPalette(
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

	items := []string{
		"Gear Calculator",
		"Crank Angle Calculator",
		"Wheel Calculator",
		"Dyno",
		"Piston Length Calculator",
		"Settings",
		"Updates",
		"About",
		"Quit",
	}

	titleStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	subtitleStyle :=
		lipgloss.NewStyle().
			Foreground(p.muted)

	itemStyle :=
		lipgloss.NewStyle().
			Foreground(p.text)

	selectedStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	var lines []string

	lines =
		append(
			lines,
			"",
			titleStyle.Render(
				"PC MULTITOOL",
			),
			"",
			subtitleStyle.Render(
				"Plane Crazy Engineering Tools",
			),
			"",
		)

	for i, item := range items {

		prefix := "  "
		style := itemStyle
		text := item

		if i == m.homeIndex {

			prefix = "> "
			style =
				selectedStyle
		}

		lines =
			append(
				lines,
				style.Render(
					prefix+text,
				),
			)
	}

	lines =
		append(
			lines,
			"",
			subtitleStyle.Render(
				"[↑↓] Select    [ENTER] Open    [Q] Quit",
			),
		)

	content :=
		strings.Join(
			lines,
			"\n",
		)

	boxWidth := 56

	if boxWidth > width-4 {
		boxWidth =
			width - 4
	}

	if boxWidth < 1 {
		boxWidth = 1
	}

	box :=
		lipgloss.NewStyle().
			Width(boxWidth).
			Padding(2, 4).
			Foreground(p.text).
			Render(content)

	output :=
		lipgloss.Place(
			width,
			height,
			lipgloss.Center,
			lipgloss.Center,
			box,
		)

	view :=
		tea.NewView(
			output,
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

// ─────────────────────────────────────────────────────────────
// Main input handling
// ─────────────────────────────────────────────────────────────

func (m model) updateMain(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "e":

		m.modeTeeth()
		return m, nil

	case "r":

		m.modeCompressors()
		return m, nil

	case "c":

		m.stopEditing()
		m.customizeTool = customizeGear
		m.customIndex = 0
		m.page = pageCustomize

		return m, nil

	case "pageup":

		m.referenceScroll--

		m.referenceScroll =
			m.clampReferenceScroll()

		return m, nil

	case "pagedown":

		m.referenceScroll++

		m.referenceScroll =
			m.clampReferenceScroll()

		return m, nil

	case "esc":

		m.stopEditing()
		m.page = pageHome

		return m, nil
	}

	var cmd tea.Cmd

	if m.teethInput.Focused() {

		m.teethInput,
			cmd =
			m.teethInput.Update(msg)

		if m.teethInput.Value() != "" {
			m.updateAutomaticCompressor()
		}

		return m, cmd
	}

	if m.compressorInput.Focused() {

		m.compressorInput,
			cmd =
			m.compressorInput.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m *model) modeTeeth() {

	m.compressorInput.Blur()
	m.crankCylinderInput.Blur()
	m.wheelTireInput.Blur()

	m.teethInput.Focus()
	m.teethInput.CursorEnd()
}

func (m *model) modeCompressors() {

	m.teethInput.Blur()
	m.crankCylinderInput.Blur()
	m.wheelTireInput.Blur()

	m.compressorInput.Focus()
	m.compressorInput.CursorEnd()
}

func (m *model) stopEditing() {

	m.teethInput.Blur()
	m.compressorInput.Blur()
	m.crankCylinderInput.Blur()
	m.wheelTireInput.Blur()
	m.pistonDistanceInput.Blur()
	m.pistonAmountInput.Blur()
	m.pistonValueInput.Blur()

	m.referenceSmallestInput.Blur()
	m.referenceMaximumInput.Blur()

	m.setupExportInput.Blur()

	for i := range m.dynoPoints {

		m.dynoPoints[i].SPSInput.Blur()
		m.dynoPoints[i].TorqueInput.Blur()
	}
}

func (m *model) updateAutomaticCompressor() {

	teeth,
		err :=
		strconv.Atoi(
			m.teethInput.Value(),
		)

	if err != nil ||
		teeth < 3 {

		return
	}

	_, _, offset :=
		Calculate(teeth)

	m.compressorInput.SetValue(
		strconv.Itoa(
			AutoCompressors(
				offset,
			),
		),
	)
}

func (m model) currentTeeth() int {

	value,
		err :=
		strconv.Atoi(
			m.teethInput.Value(),
		)

	if err != nil ||
		value < 3 {

		return 6
	}

	return value
}

func (m model) currentCompressors() int {

	value,
		err :=
		strconv.Atoi(
			m.compressorInput.Value(),
		)

	if err != nil ||
		value <= 0 {

		return 1
	}

	return value
}

// ─────────────────────────────────────────────────────────────
// Customization
// ─────────────────────────────────────────────────────────────

func (m model) updateCustomize(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	items :=
		m.customizeItems()

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.customIndex--

		if m.customIndex < 0 {
			m.customIndex =
				len(items) - 1
		}

	case "down":

		m.customIndex++

		if m.customIndex >=
			len(items) {

			m.customIndex = 0
		}

	case "enter":

		switch m.customizeTool {

		case customizeGear:

			switch m.customIndex {

			case 0:

				m.page = pageAppearance
				m.appearanceIndex = 0

			case 1:

				m.page = pageLayout
				m.layoutIndex = 0

			case 2:

				m.page = pageReference
				m.referenceIndex = 0

			case 3:

				m.page = pageCalculator
				m.calculatorIndex = 0

			case 4:

				m.page = pageReset
				m.resetIndex = 1

			case 5:

				m.page =
					customizeToolPage(
						m.customizeTool,
					)
			}

		case customizeCrank,
			customizeWheel,
			customizeDyno,
			customizePiston:

			switch m.customIndex {

			case 0:

				m.page = pageAppearance
				m.appearanceIndex = 0

			case 1:

				m.page = pageLayout
				m.layoutIndex = 0

			case 2:

				m.page = pageReset
				m.resetIndex = 1

			case 3:

				m.page =
					customizeToolPage(
						m.customizeTool,
					)
			}
		}

	case "esc":

		m.page =
			customizeToolPage(
				m.customizeTool,
			)
	}

	return m, nil
}

func (m model) customizeItems() []string {

	switch m.customizeTool {

	case customizeGear:

		return []string{
			"Appearance",
			"Layout",
			"Reference Chart",
			"Calculator",
			"Reset to Defaults",
			"Back",
		}

	case customizeCrank,
		customizeWheel,
		customizeDyno,
		customizePiston:

		return []string{
			"Appearance",
			"Layout",
			"Reset to Defaults",
			"Back",
		}

	default:

		return []string{
			"Back",
		}
	}
}

func (m model) updateToolCustomizeSelect(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	items :=
		[]string{
			"Gear Calculator",
			"Crank Angle Calculator",
			"Wheel Calculator",
			"Dyno",
			"Piston Length Calculator",
			"Back",
		}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.customToolIndex--

		if m.customToolIndex < 0 {
			m.customToolIndex =
				len(items) - 1
		}

	case "down":

		m.customToolIndex++

		if m.customToolIndex >=
			len(items) {

			m.customToolIndex = 0
		}

	case "enter":

		switch m.customToolIndex {

		case 0:
			m.customizeTool =
				customizeGear

		case 1:
			m.customizeTool =
				customizeCrank

		case 2:
			m.customizeTool =
				customizeWheel

		case 3:
			m.customizeTool =
				customizeDyno

		case 4:
			m.customizeTool =
				customizePiston

		case 5:
			m.page = pageSettings
			m.settingsIndex = 1

			return m, nil
		}

		m.customIndex = 0
		m.page = pageCustomize

	case "esc":

		m.page = pageSettings
		m.settingsIndex = 1
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// Appearance
// ─────────────────────────────────────────────────────────────
type updateCheckMsg struct {
	info updateInfo
	err  error
}

type updateDownloadMsg struct {
	path string
	err  error
}

func (m model) updateUpdates(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	items :=
		[]string{
			"Check for updates",
		}

	if m.updateInfo != nil &&
		isNewerVersion(
			m.updateInfo.LatestVersion,
			m.updateInfo.CurrentVersion,
		) {

		items =
			append(
				items,
				"Update",
			)
	}

	items =
		append(
			items,
			"Back",
		)

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.updatesIndex--

		if m.updatesIndex < 0 {
			m.updatesIndex =
				len(items) - 1
		}

	case "down":

		m.updatesIndex++

		if m.updatesIndex >= len(items) {
			m.updatesIndex = 0
		}

	case "enter":

		switch items[m.updatesIndex] {

		case "Check for updates":

			if m.updateChecking {
				return m, nil
			}

			m.updateChecking = true
			m.updateStatus =
				"Checking for updates..."

			return m, func() tea.Msg {
				info, err :=
					getLatestUpdate()

				return updateCheckMsg{
					info: info,
					err:  err,
				}
			}

		case "Update":

			if m.updateInfo == nil ||
				!isNewerVersion(
					m.updateInfo.LatestVersion,
					m.updateInfo.CurrentVersion,
				) {

				return m, nil
			}

			if m.updateChecking {
				return m, nil
			}

			m.updateChecking = true
			m.updateStatus =
				"Downloading and verifying..."

			asset :=
				m.updateInfo.Asset

			return m, func() tea.Msg {

				path, err :=
					downloadUpdate(asset)

				return updateDownloadMsg{
					path: path,
					err:  err,
				}
			}

		case "Back":

			m.page = pageHome
			m.homeIndex = 6
		}

	case "esc":

		m.page = pageHome
		m.homeIndex = 6
	}

	return m, nil
}

func (m model) viewUpdates() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	items :=
		[]string{
			"Check for updates",
		}

	if m.updateInfo != nil &&
		isNewerVersion(
			m.updateInfo.LatestVersion,
			m.updateInfo.CurrentVersion,
		) {

		items =
			append(
				items,
				"Update",
			)
	}

	items =
		append(
			items,
			"Back",
		)

	title := "UPDATES"

	if m.updateInfo != nil {
		title = fmt.Sprintf(
			"UPDATES  •  CURRENT %s  •  LATEST %s",
			m.updateInfo.CurrentVersion,
			m.updateInfo.LatestVersion,
		)
	}

	if m.updateStatus != "" {
		title += "  •  " + m.updateStatus
	}

	return m.renderMenu(
		title,
		items,
		m.updatesIndex,
		p,
	)
}

func (m model) updateSettings(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	items :=
		[]string{
			"Appearance",
			"Layout",
			"Update channel  •  " +
				updateChannelName(m.cfg.UpdateChannel),
			"Automatic updates  •  " +
				map[bool]string{
					true:  "ON",
					false: "OFF",
				}[m.cfg.AutomaticUpdates],
			"Back",
		}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":
		m.settingsIndex--

		if m.settingsIndex < 0 {
			m.settingsIndex =
				len(items) - 1
		}

	case "down":
		m.settingsIndex++

		if m.settingsIndex >= len(items) {
			m.settingsIndex = 0
		}

	case "left":

		switch m.settingsIndex {

		case 2:
			m.cycleUpdateChannel(-1)

		case 3:
			m.toggleAutomaticUpdates()
		}

	case "right":

		switch m.settingsIndex {

		case 2:
			m.cycleUpdateChannel(1)

		case 3:
			m.toggleAutomaticUpdates()
		}

	case "enter":

		switch m.settingsIndex {

		case 0:
			m.appearanceIndex = 0
			m.page = pageAppearance

		case 1:
			m.customToolIndex = 0
			m.customIndex = 0
			m.page = pageToolCustomizeSelect

		case 2:
			m.cycleUpdateChannel(1)

		case 3:
			m.toggleAutomaticUpdates()

		case 4:
			m.page = pageHome
			m.homeIndex = 5
		}

	case "esc":
		m.page = pageHome
		m.homeIndex = 5
	}

	return m, nil
}

func (m model) viewSettings() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	items :=
		[]string{
			"Appearance",
			"Layout",
			"Update channel  •  " +
				updateChannelName(m.cfg.UpdateChannel),
			"Automatic updates  •  " +
				map[bool]string{
					true:  "ON",
					false: "OFF",
				}[m.cfg.AutomaticUpdates],
			"Back",
		}

	return m.renderMenu(
		"SETTINGS",
		items,
		m.settingsIndex,
		p,
	)
}

func (m model) updateAppearance(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	items :=
		[]string{
			"Theme",
			"Accent",
			"Background",
			"Panel Transparency",
			"Back",
		}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.appearanceIndex--

		if m.appearanceIndex < 0 {
			m.appearanceIndex =
				len(items) - 1
		}

	case "down":

		m.appearanceIndex++

		if m.appearanceIndex >=
			len(items) {

			m.appearanceIndex = 0
		}

	case "enter":

		switch m.appearanceIndex {

		case 0:

			m.updateThemeIndex()
			m.page = pageThemes

		case 1:

			m.updateAccentIndex()
			m.page = pageAccents

		case 2:

			if m.cfg.Appearance.Background ==
				"theme" {

				m.cfg.Appearance.Background =
					"transparent"

			} else {

				m.cfg.Appearance.Background =
					"theme"
			}

			m.saveSettings()

		case 3:

			m.transparencyIndex = 0
			m.page = pageTransparency

		case 4:

			m.page = pageCustomize
		}

	case "esc":

		m.page = pageCustomize
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// Themes
// ─────────────────────────────────────────────────────────────

func (m model) updateThemes(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	themes := []string{
		"Catppuccin",
		"Nord",
		"Gruvbox",
		"Solid",
	}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.themeIndex--

		if m.themeIndex < 0 {
			m.themeIndex = len(themes) - 1
		}

	case "down":

		m.themeIndex++

		if m.themeIndex >= len(themes) {
			m.themeIndex = 0
		}

	case "enter":

		switch m.themeIndex {

		case 0:
			m.catppuccinFlavorIndex = m.currentCatppuccinFlavorIndex()
			m.page = pageFlavors
			return m, nil

		case 1:
			m.theme = "nord"

		case 2:
			m.theme = "gruvbox"

		case 3:
			m.theme = "solid"
		}

		m.cfg.Appearance.Theme = m.theme
		m.saveSettings()
		m.updateAccentIndex()
		m.page = pageAccents

	case "esc":

		m.page = pageAppearance
	}

	return m, nil
}

func (m model) catppuccinFlavorKeys() []string {
	return []string{
		"catppuccin-latte",
		"catppuccin-frappe",
		"catppuccin-macchiato",
		"catppuccin-mocha",
	}
}

func (m model) catppuccinFlavorNames() []string {
	return []string{
		"Latte",
		"Frappé",
		"Macchiato",
		"Mocha",
	}
}

func (m model) currentCatppuccinFlavorIndex() int {

	keys := m.catppuccinFlavorKeys()

	for i, key := range keys {
		if m.theme == key {
			return i
		}
	}

	return 3
}

func (m model) updateCatppuccinFlavors(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	names := m.catppuccinFlavorNames()
	keys := m.catppuccinFlavorKeys()

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.catppuccinFlavorIndex--

		if m.catppuccinFlavorIndex < 0 {
			m.catppuccinFlavorIndex = len(names) - 1
		}

	case "down":

		m.catppuccinFlavorIndex++

		if m.catppuccinFlavorIndex >= len(names) {
			m.catppuccinFlavorIndex = 0
		}

	case "enter":

		m.theme = keys[m.catppuccinFlavorIndex]
		m.cfg.Appearance.Theme = m.theme
		m.saveSettings()

		m.updateAccentIndex()
		m.page = pageAccents

	case "esc":

		m.page = pageThemes
	}

	return m, nil
}

func (m model) updateAccents(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.accentIndex--

		if m.accentIndex < 0 {
			m.accentIndex =
				len(accentKeys) - 1
		}

	case "down":

		m.accentIndex++

		if m.accentIndex >=
			len(accentKeys) {

			m.accentIndex = 0
		}

	case "enter":

		m.accent =
			accentKeys[m.accentIndex]

		m.cfg.Appearance.Accent =
			m.accent

		m.saveSettings()
		m.page = pageAppearance

	case "esc":

		m.page = pageAppearance
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// Transparency
// ─────────────────────────────────────────────────────────────

func (m model) updateTransparency(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	items :=
		[]string{
			"Top Bar",
			"Reference Panel",
			"Calculator Panel",
			"Bottom Bar",
			"Back",
		}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.transparencyIndex--

		if m.transparencyIndex < 0 {
			m.transparencyIndex =
				len(items) - 1
		}

	case "down":

		m.transparencyIndex++

		if m.transparencyIndex >=
			len(items) {

			m.transparencyIndex = 0
		}

	case "space", "enter":

		switch m.transparencyIndex {

		case 0:

			m.cfg.Appearance.TopBarTransparent =
				!m.cfg.Appearance.TopBarTransparent

		case 1:

			m.cfg.Appearance.ReferenceTransparent =
				!m.cfg.Appearance.ReferenceTransparent

		case 2:

			m.cfg.Appearance.CalculatorTransparent =
				!m.cfg.Appearance.CalculatorTransparent

		case 3:

			m.cfg.Appearance.BottomBarTransparent =
				!m.cfg.Appearance.BottomBarTransparent

		case 4:

			m.page = pageAppearance
			return m, nil
		}

		m.saveSettings()

	case "esc":

		m.page = pageAppearance
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// Layout
// ─────────────────────────────────────────────────────────────

func (m model) updateLayout(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.layoutItems()

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":
		m.layoutIndex--
		if m.layoutIndex < 0 {
			m.layoutIndex = len(items) - 1
		}

	case "down":
		m.layoutIndex++
		if m.layoutIndex >= len(items) {
			m.layoutIndex = 0
		}

	case "left", "right":
		if m.customizeTool == customizeGear {
			if m.layoutIndex == 6 {
				if msg.String() == "left" {
					m.cycleReferenceWidth(-1)
				} else {
					m.cycleReferenceWidth(1)
				}
			} else if m.layoutIndex == 7 {
				m.toggleOrder()
			}
		}

	case "enter":
		switch m.customizeTool {

		case customizeGear:
			switch m.layoutIndex {
			case 0:
				m.setLayout("automatic")
			case 1:
				m.setLayout("calculator-left")
			case 2:
				m.setLayout("calculator-right")
			case 3:
				m.setLayout("calculator-only")
			case 4:
				m.setLayout("reference-only")
			case 5:
				m.setLayout("stacked")
			case 6:
				m.cycleReferenceWidth(1)
			case 7:
				m.toggleOrder()
			case 8:
				m.page = pageCustomize
			}

		case customizeCrank:
			switch m.layoutIndex {
			case 0:
				m.cfg.Crank.Layout = "results-left"
				m.saveSettings()
			case 1:
				m.cfg.Crank.Layout = "results-right"
				m.saveSettings()
			case 2:
				m.page = pageCustomize
			}

		case customizeWheel:
			switch m.layoutIndex {
			case 0:
				m.cfg.Wheel.ResultSide = "left"
				m.saveSettings()
			case 1:
				m.cfg.Wheel.ResultSide = "right"
				m.saveSettings()
			case 2:
				m.page = pageCustomize
			}

		case customizeDyno:
			switch m.layoutIndex {
			case 0:
				m.cfg.Dyno.GraphSide = "left"
				m.saveSettings()
			case 1:
				m.cfg.Dyno.GraphSide = "right"
				m.saveSettings()
			case 2:
				m.page = pageCustomize
			}

		case customizePiston:
			switch m.layoutIndex {
			case 0:
				m.cfg.Piston.ResultSide = "left"
				m.saveSettings()

			case 1:
				m.cfg.Piston.ResultSide = "right"
				m.saveSettings()

			case 2:
				m.page = pageCustomize
			}
		}

	case "esc":
		m.page = pageCustomize
	}

	return m, nil
}

func (m model) layoutItems() []string {
	switch m.customizeTool {

	case customizeGear:
		return []string{
			"Automatic",
			"Calculator left",
			"Calculator right",
			"Calculator only",
			"Reference only",
			"Stacked",
			"Reference width",
			"Stacked order",
			"Back",
		}

	case customizeCrank:
		return []string{
			"Results left" + selectedSuffix(
				m.cfg.Crank.Layout == "results-left",
			),
			"Results right" + selectedSuffix(
				m.cfg.Crank.Layout == "results-right",
			),
			"Back",
		}

	case customizeWheel:
		return []string{
			"Results left" + selectedSuffix(
				m.cfg.Wheel.ResultSide == "left",
			),
			"Results right" + selectedSuffix(
				m.cfg.Wheel.ResultSide == "right",
			),
			"Back",
		}

	case customizeDyno:
		return []string{
			"Graph left" + selectedSuffix(
				m.cfg.Dyno.GraphSide == "left",
			),
			"Graph right" + selectedSuffix(
				m.cfg.Dyno.GraphSide == "right",
			),
			"Back",
		}

	case customizePiston:
		return []string{
			"Results left" + selectedSuffix(
				m.cfg.Piston.ResultSide == "left",
			),
			"Results right" + selectedSuffix(
				m.cfg.Piston.ResultSide == "right",
			),
			"Back",
		}

	default:
		return []string{"Back"}
	}
}

func (m *model) setLayout(
	mode string,
) {

	m.cfg.Layout.Mode =
		mode

	m.referenceScroll =
		0

	m.saveSettings()
}

func (m *model) toggleOrder() {

	if m.cfg.Layout.Order ==
		"reference-first" {

		m.cfg.Layout.Order =
			"calculator-first"

	} else {

		m.cfg.Layout.Order =
			"reference-first"
	}

	m.referenceScroll =
		m.clampReferenceScroll()

	m.saveSettings()
}

func (m *model) cycleReferenceWidth(
	delta int,
) {

	values :=
		[]string{
			"balanced",
			"50",
			"60",
			"70",
			"80",
		}

	current := 0

	for i, value := range values {

		if value ==
			m.cfg.Layout.ReferenceWidth {

			current = i
			break
		}
	}

	current += delta

	for current < 0 {
		current += len(values)
	}

	current %=
		len(values)

	m.cfg.Layout.ReferenceWidth =
		values[current]

	m.referenceScroll =
		m.clampReferenceScroll()

	m.saveSettings()
}

func (m model) panelWidths(
	width int,
) (int, int) {

	left :=
		int(
			float64(width) *
				0.70,
		)

	switch m.cfg.Layout.ReferenceWidth {

	case "50":

		left =
			width / 2

	case "60":

		left =
			int(
				float64(width) *
					0.60,
			)

	case "70":

		left =
			int(
				float64(width) *
					0.70,
			)

	case "80":

		left =
			int(
				float64(width) *
					0.80,
			)

	case "balanced":

		left =
			int(
				float64(width) *
					0.70,
			)
	}

	if left < 1 {
		left = 1
	}

	right :=
		width -
			left -
			1

	if right < 1 {
		right = 1
	}

	return left, right
}

func (m model) effectiveLayout() string {

	mode :=
		m.cfg.Layout.Mode

	if mode != "automatic" {
		return mode
	}

	if m.width < 80 {
		return "calculator-only"
	}

	return "calculator-right"
}

// ─────────────────────────────────────────────────────────────
// Reference viewport
// ─────────────────────────────────────────────────────────────

func (m model) referencePanelHeight() int {

	contentHeight :=
		m.height - 4

	if contentHeight < 1 {
		contentHeight = 1
	}

	mode :=
		m.effectiveLayout()

	if mode != "stacked" {
		return contentHeight
	}

	calculatorHeight :=
		5

	if contentHeight < 8 {
		calculatorHeight =
			contentHeight / 2
	}

	if calculatorHeight < 1 {
		calculatorHeight = 1
	}

	referenceHeight :=
		contentHeight -
			calculatorHeight

	if referenceHeight < 1 {
		referenceHeight = 1
	}

	return referenceHeight
}

func (m model) referenceVisibleRows(
	panelHeight int,
) int {

	// Header uses:
	// title
	// blank
	// column header
	// divider
	// blank
	rows :=
		panelHeight - 5

	if rows < 1 {
		rows = 1
	}

	totalRows :=
		m.cfg.Reference.MaximumGear -
			m.cfg.Reference.SmallestGear +
			1

	if totalRows < 1 {
		totalRows = 1
	}

	// Reserve one line for scroll indicators.
	if totalRows > rows {
		rows--
	}

	if rows < 1 {
		rows = 1
	}

	return rows
}

func (m model) referenceMaxScroll() int {

	panelHeight :=
		m.referencePanelHeight()

	visibleRows :=
		m.referenceVisibleRows(
			panelHeight,
		)

	totalRows :=
		m.cfg.Reference.MaximumGear -
			m.cfg.Reference.SmallestGear +
			1

	if totalRows < 1 {
		return 0
	}

	maxScroll :=
		totalRows -
			visibleRows

	if maxScroll < 0 {
		maxScroll = 0
	}

	return maxScroll
}

func (m model) clampReferenceScroll() int {

	maxScroll :=
		m.referenceMaxScroll()

	if m.referenceScroll < 0 {
		return 0
	}

	if m.referenceScroll > maxScroll {
		return maxScroll
	}

	return m.referenceScroll
}

func (m model) referenceMouseArea(
	x int,
	y int,
) bool {

	if !m.cfg.Reference.Enabled {
		return false
	}

	mode :=
		m.effectiveLayout()

	contentHeight :=
		m.height - 4

	if contentHeight < 1 {
		contentHeight = 1
	}

	// Main content starts after the two-line top bar.
	if y < 2 ||
		y >= 2+contentHeight {

		return false
	}

	switch mode {

	case "reference-only":

		return true

	case "calculator-only":

		return false

	case "stacked":

		calculatorHeight :=
			5

		if contentHeight < 8 {
			calculatorHeight =
				contentHeight / 2
		}

		if calculatorHeight < 1 {
			calculatorHeight = 1
		}

		referenceHeight :=
			contentHeight -
				calculatorHeight

		if referenceHeight < 1 {
			referenceHeight = 1
		}

		if m.cfg.Layout.Order ==
			"calculator-first" {

			referenceTop :=
				2 +
					calculatorHeight

			return y >= referenceTop &&
				y <
					referenceTop+
						referenceHeight
		}

		return y >= 2 &&
			y <
				2+
					referenceHeight

	case "calculator-left":

		leftWidth,
			_ :=
			m.panelWidths(
				m.width,
			)

		return x >= leftWidth+1 &&
			x < m.width

	default:

		leftWidth,
			_ :=
			m.panelWidths(
				m.width,
			)

		return x >= 0 &&
			x < leftWidth
	}
}

// ─────────────────────────────────────────────────────────────
// Reference settings
// ─────────────────────────────────────────────────────────────

func (m model) updateReference(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	items :=
		[]string{
			"Enabled",
			"Smallest Gear",
			"Maximum Gear",
			"Teeth",
			"Full Angle",
			"Half Angle",
			"Offset",
			"Compressor Value",
			"Compressors",
			"Back",
		}

	if m.referenceSmallestInput.Focused() {

		switch msg.String() {

		case "enter":

			value,
				err :=
				strconv.Atoi(
					strings.TrimSpace(
						m.referenceSmallestInput.Value(),
					),
				)

			if err == nil &&
				value >= 1 {

				if value >
					m.cfg.Reference.MaximumGear {

					value =
						m.cfg.Reference.MaximumGear
				}

				m.cfg.Reference.SmallestGear =
					value

				m.referenceSmallestInput.SetValue(
					strconv.Itoa(value),
				)

				m.referenceScroll =
					m.clampReferenceScroll()

				m.saveSettings()
			}

			m.referenceSmallestInput.Blur()

			return m, nil

		case "esc":

			m.referenceSmallestInput.SetValue(
				strconv.Itoa(
					m.cfg.Reference.SmallestGear,
				),
			)

			m.referenceSmallestInput.Blur()

			return m, nil
		}

		var cmd tea.Cmd

		m.referenceSmallestInput,
			cmd =
			m.referenceSmallestInput.Update(msg)

		return m, cmd
	}

	if m.referenceMaximumInput.Focused() {

		switch msg.String() {

		case "enter":

			value,
				err :=
				strconv.Atoi(
					strings.TrimSpace(
						m.referenceMaximumInput.Value(),
					),
				)

			if err == nil &&
				value >= 1 {

				if value <
					m.cfg.Reference.SmallestGear {

					value =
						m.cfg.Reference.SmallestGear
				}

				m.cfg.Reference.MaximumGear =
					value

				m.referenceMaximumInput.SetValue(
					strconv.Itoa(value),
				)

				m.referenceScroll =
					m.clampReferenceScroll()

				m.saveSettings()
			}

			m.referenceMaximumInput.Blur()

			return m, nil

		case "esc":

			m.referenceMaximumInput.SetValue(
				strconv.Itoa(
					m.cfg.Reference.MaximumGear,
				),
			)

			m.referenceMaximumInput.Blur()

			return m, nil
		}

		var cmd tea.Cmd

		m.referenceMaximumInput,
			cmd =
			m.referenceMaximumInput.Update(msg)

		return m, cmd
	}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.referenceIndex--

		if m.referenceIndex < 0 {
			m.referenceIndex =
				len(items) - 1
		}

	case "down":

		m.referenceIndex++

		if m.referenceIndex >=
			len(items) {

			m.referenceIndex = 0
		}

	case "enter", "space":

		switch m.referenceIndex {

		case 0:

			m.cfg.Reference.Enabled =
				!m.cfg.Reference.Enabled

			m.referenceScroll = 0
			m.saveSettings()

		case 1:

			m.referenceMaximumInput.Blur()

			m.referenceSmallestInput.SetValue(
				strconv.Itoa(
					m.cfg.Reference.SmallestGear,
				),
			)

			m.referenceSmallestInput.Focus()
			m.referenceSmallestInput.CursorEnd()

		case 2:

			m.referenceSmallestInput.Blur()

			m.referenceMaximumInput.SetValue(
				strconv.Itoa(
					m.cfg.Reference.MaximumGear,
				),
			)

			m.referenceMaximumInput.Focus()
			m.referenceMaximumInput.CursorEnd()

		case 3:

			m.cfg.Reference.Teeth =
				!m.cfg.Reference.Teeth

			m.saveSettings()

		case 4:

			m.cfg.Reference.FullAngle =
				!m.cfg.Reference.FullAngle

			m.saveSettings()

		case 5:

			m.cfg.Reference.HalfAngle =
				!m.cfg.Reference.HalfAngle

			m.saveSettings()

		case 6:

			m.cfg.Reference.Offset =
				!m.cfg.Reference.Offset

			m.saveSettings()

		case 7:

			m.cfg.Reference.CompressorValue =
				!m.cfg.Reference.CompressorValue

			m.saveSettings()

		case 8:

			m.cfg.Reference.Compressors =
				!m.cfg.Reference.Compressors

			m.saveSettings()

		case 9:

			m.page =
				pageCustomize
		}

	case "esc":

		m.referenceSmallestInput.Blur()
		m.referenceMaximumInput.Blur()

		m.page =
			pageCustomize
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// Reference settings mouse handling
// ─────────────────────────────────────────────────────────────

func (m model) handleReferenceMouse(
	msg tea.MouseClickMsg,
) (tea.Model, tea.Cmd) {

	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	/*
		Reference Chart menu:

		0 Enabled
		1 Smallest Gear
		2 Maximum Gear
		3 Teeth
		4 Full Angle
		5 Half Angle
		6 Offset
		7 Compressor Value
		8 Compressors
		9 Back
	*/

	height :=
		m.height

	width :=
		m.width

	if height < 1 {
		height = 1
	}

	if width < 1 {
		width = 1
	}

	boxHeight :=
		20

	if boxHeight >
		height-2 {

		boxHeight =
			height - 2
	}

	if boxHeight < 1 {
		boxHeight = 1
	}

	outerHeight :=
		boxHeight +
			4 +
			2

	boxTop :=
		(height -
			outerHeight) /
			2

	firstItemY :=
		boxTop +
			1 +
			2 +
			2

	index :=
		msg.Y -
			firstItemY

	if index < 0 ||
		index > 9 {

		return m, nil
	}

	m.referenceIndex =
		index

	switch index {

	case 0:

		m.cfg.Reference.Enabled =
			!m.cfg.Reference.Enabled

		m.referenceScroll = 0
		m.saveSettings()

	case 1:

		m.referenceMaximumInput.Blur()

		m.referenceSmallestInput.SetValue(
			strconv.Itoa(
				m.cfg.Reference.SmallestGear,
			),
		)

		m.referenceSmallestInput.Focus()
		m.referenceSmallestInput.CursorEnd()

	case 2:

		m.referenceSmallestInput.Blur()

		m.referenceMaximumInput.SetValue(
			strconv.Itoa(
				m.cfg.Reference.MaximumGear,
			),
		)

		m.referenceMaximumInput.Focus()
		m.referenceMaximumInput.CursorEnd()

	case 3:

		m.cfg.Reference.Teeth =
			!m.cfg.Reference.Teeth

		m.saveSettings()

	case 4:

		m.cfg.Reference.FullAngle =
			!m.cfg.Reference.FullAngle

		m.saveSettings()

	case 5:

		m.cfg.Reference.HalfAngle =
			!m.cfg.Reference.HalfAngle

		m.saveSettings()

	case 6:

		m.cfg.Reference.Offset =
			!m.cfg.Reference.Offset

		m.saveSettings()

	case 7:

		m.cfg.Reference.CompressorValue =
			!m.cfg.Reference.CompressorValue

		m.saveSettings()

	case 8:

		m.cfg.Reference.Compressors =
			!m.cfg.Reference.Compressors

		m.saveSettings()

	case 9:

		m.referenceSmallestInput.Blur()
		m.referenceMaximumInput.Blur()

		m.page =
			pageCustomize
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// Calculator settings
// ─────────────────────────────────────────────────────────────

func (m model) updateCalculator(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	items :=
		[]string{
			"Enabled",
			"Number of Teeth",
			"Compressors",
			"Full Angle",
			"Half Angle",
			"Offset",
			"Compressor Value",
			"Warnings",
			"Back",
		}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.calculatorIndex--

		if m.calculatorIndex < 0 {
			m.calculatorIndex =
				len(items) - 1
		}

	case "down":

		m.calculatorIndex++

		if m.calculatorIndex >=
			len(items) {

			m.calculatorIndex = 0
		}

	case "space", "enter":

		switch m.calculatorIndex {

		case 0:

			m.cfg.Calculator.Enabled =
				!m.cfg.Calculator.Enabled

		case 1:

			m.cfg.Calculator.Teeth =
				!m.cfg.Calculator.Teeth

		case 2:

			m.cfg.Calculator.Compressors =
				!m.cfg.Calculator.Compressors

		case 3:

			m.cfg.Calculator.FullAngle =
				!m.cfg.Calculator.FullAngle

		case 4:

			m.cfg.Calculator.HalfAngle =
				!m.cfg.Calculator.HalfAngle

		case 5:

			m.cfg.Calculator.Offset =
				!m.cfg.Calculator.Offset

		case 6:

			m.cfg.Calculator.CompressorValue =
				!m.cfg.Calculator.CompressorValue

		case 7:

			m.cfg.Calculator.Warnings =
				!m.cfg.Calculator.Warnings

		case 8:

			m.page =
				pageCustomize

			return m, nil
		}

		m.saveSettings()

	case "esc":

		m.page =
			pageCustomize
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// Reset
// ─────────────────────────────────────────────────────────────

func (m model) updateReset(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up",
		"down",
		"left",
		"right":

		m.resetIndex ^= 1

	case "enter":

		if m.resetIndex == 0 {

			m.cfg =
				defaultConfig()

			m.theme =
				m.cfg.Appearance.Theme

			m.accent =
				m.cfg.Appearance.Accent

			m.updateThemeIndex()
			m.updateAccentIndex()

			m.referenceScroll = 0

			m.referenceSmallestInput.SetValue(
				strconv.Itoa(
					m.cfg.Reference.SmallestGear,
				),
			)

			m.referenceMaximumInput.SetValue(
				strconv.Itoa(
					m.cfg.Reference.MaximumGear,
				),
			)

			m.saveSettings()
		}

		m.page =
			pageCustomize

	case "esc":

		m.page =
			pageCustomize
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// Settings helpers
// ─────────────────────────────────────────────────────────────

func (m *model) saveSettings() {

	saveConfig(m.cfg)

	m.theme =
		m.cfg.Appearance.Theme

	m.accent =
		m.cfg.Appearance.Accent
}

func (m *model) updateThemeIndex() {

	for i, key := range themeKeys {

		if key ==
			m.theme {

			m.themeIndex =
				i

			return
		}
	}

	m.themeIndex =
		0
}

func (m *model) updateAccentIndex() {

	for i, key := range accentKeys {

		if key ==
			m.accent {

			m.accentIndex =
				i

			return
		}
	}

	m.accentIndex =
		0
}

// ─────────────────────────────────────────────────────────────
// Mouse
// ─────────────────────────────────────────────────────────────

func (m model) menuMouseIndex(
	msg tea.MouseClickMsg,
	itemCount int,
) (int, bool) {

	if itemCount <= 0 {
		return 0, false
	}

	width := m.width
	height := m.height

	if width < 1 {
		width = 1
	}

	if height < 1 {
		height = 1
	}

	// renderMenu:
	//
	//   vertical content:
	//     title
	//     blank
	//     items...
	//     blank
	//     controls
	//
	//   box padding: 2 top + 2 bottom
	//   box border: 1 top + 1 bottom
	//
	// Therefore:
	//
	//   boxHeight = itemCount + 10
	//
	// The first item begins 5 rows below the box top:
	//
	//   1 border
	//   2 padding
	//   1 title
	//   1 blank
	boxHeight :=
		itemCount + 10

	boxTop :=
		(height - boxHeight) / 2

	firstItemY :=
		boxTop + 5

	index :=
		msg.Y - firstItemY

	if index < 0 ||
		index >= itemCount {

		return 0, false
	}

	// renderMenu has a fixed width of 62 plus:
	// 3 left padding + 3 right padding + 2 borders.
	boxWidth := 70

	boxLeft :=
		(width - boxWidth) / 2

	if msg.X < boxLeft ||
		msg.X >= boxLeft+boxWidth {

		return 0, false
	}

	return index, true
}

// handleMenuMouse gives every renderMenu-based screen the same mouse
// behavior as its keyboard selection. This keeps mouse geometry in one place
// instead of duplicating fixed coordinates for each settings screen.
func (m model) handleMenuMouse(
	msg tea.MouseClickMsg,
) (bool, tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return false, m, nil
	}

	itemCount := 0

	switch m.page {
	case pageSettings:
		itemCount = 3
	case pageUpdates:
		itemCount = 2
		if m.updateInfo != nil && isNewerVersion(m.updateInfo.LatestVersion, m.updateInfo.CurrentVersion) {
			itemCount = 3
		}
	case pageCustomize:
		itemCount = len(m.customizeItems())
	case pageToolCustomizeSelect:
		itemCount = 6
	case pageAppearance, pageTransparency:
		itemCount = 5
	case pageThemes:
		itemCount = 4
	case pageFlavors:
		itemCount = 4
	case pageAccents:
		itemCount = len(accentKeys)
	case pageLayout:
		itemCount = len(m.layoutItems())
	case pageCalculator:
		itemCount = 9
	case pageReset:
		itemCount = 2
	default:
		return false, m, nil
	}

	index, ok := m.menuMouseIndex(msg, itemCount)
	if !ok {
		return false, m, nil
	}

	enter := tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})

	switch m.page {
	case pageSettings:
		m.settingsIndex = index
		next, cmd := m.updateSettings(enter)
		return true, next, cmd
	case pageUpdates:
		m.updatesIndex = index
		next, cmd := m.updateUpdates(enter)
		return true, next, cmd
	case pageCustomize:
		m.customIndex = index
		next, cmd := m.updateCustomize(enter)
		return true, next, cmd
	case pageToolCustomizeSelect:
		m.customToolIndex = index
		next, cmd := m.updateToolCustomizeSelect(enter)
		return true, next, cmd
	case pageAppearance:
		m.appearanceIndex = index
		next, cmd := m.updateAppearance(enter)
		return true, next, cmd
	case pageThemes:
		m.themeIndex = index
		next, cmd := m.updateThemes(enter)
		return true, next, cmd
	case pageFlavors:
		m.catppuccinFlavorIndex = index
		next, cmd := m.updateCatppuccinFlavors(enter)
		return true, next, cmd
	case pageAccents:
		m.accentIndex = index
		next, cmd := m.updateAccents(enter)
		return true, next, cmd
	case pageTransparency:
		m.transparencyIndex = index
		next, cmd := m.updateTransparency(enter)
		return true, next, cmd
	case pageLayout:
		m.layoutIndex = index
		next, cmd := m.updateLayout(enter)
		return true, next, cmd
	case pageCalculator:
		m.calculatorIndex = index
		next, cmd := m.updateCalculator(enter)
		return true, next, cmd
	case pageReset:
		m.resetIndex = index
		next, cmd := m.updateReset(enter)
		return true, next, cmd
	}

	return false, m, nil
}

func (m model) handleMouse(
	msg tea.MouseClickMsg,
) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	if m.page == pageHome {

		items := []string{
			"Gear Calculator",
			"Crank Angle Calculator",
			"Wheel Calculator",
			"Dyno",
			"Piston Length Calculator",
			"Settings",
			"Updates",
			"About",
			"Quit",
		}

		// Home uses viewHome(), not renderMenu(), so it has
		// different padding/height geometry.
		boxHeight := 20
		boxTop := (m.height - boxHeight) / 2

		firstItemY :=
			boxTop + 7

		index :=
			msg.Y - firstItemY

		if index >= 0 &&
			index < len(items) {

			return m.openHomeItem(index)
		}

		return m, nil
	}

	if m.page == pageSetup {
		return m.handleSetupMouse(msg)
	}

	if m.page == pageReference {
		return m.handleReferenceMouse(msg)
	}

	if m.page == pageCrank {
		return m.handleCrankMouse(msg)
	}

	if m.page == pageWheel {
		return m.handleWheelMouse(msg)
	}

	if m.page == pageDyno {
		return m.handleDynoMouse(msg)
	}
	if m.page == pagePiston {
		return m.handlePistonMouse(msg)
	}

	if handled, next, cmd := m.handleMenuMouse(msg); handled {
		return next, cmd
	}

	if msg.Button !=
		tea.MouseLeft {

		return m, nil
	}

	if m.page != pageMain {
		return m, nil
	}

	width :=
		m.width

	if width < 1 {
		width = 1
	}

	mode :=
		m.effectiveLayout()

	if mode == "reference-only" ||
		!m.cfg.Calculator.Enabled {

		m.stopEditing()

		return m, nil
	}

	if mode == "stacked" {

		contentHeight :=
			m.height - 4

		if contentHeight < 1 {
			contentHeight = 1
		}

		calculatorHeight :=
			5

		if contentHeight < 8 {
			calculatorHeight =
				contentHeight / 2
		}

		if calculatorHeight < 1 {
			calculatorHeight = 1
		}

		referenceHeight :=
			contentHeight -
				calculatorHeight

		if referenceHeight < 1 {
			referenceHeight = 1
		}

		calculatorTop :=
			2 +
				referenceHeight

		if m.cfg.Layout.Order ==
			"calculator-first" {

			calculatorTop =
				2
		}

		fieldY :=
			calculatorTop

		if msg.Y ==
			fieldY {

			x := 2

			if m.cfg.Calculator.Teeth {

				if msg.X >= x &&
					msg.X < x+22 {

					m.modeTeeth()

					return m, nil
				}

				x += 22
			}

			if m.cfg.Calculator.Compressors {

				if msg.X >= x &&
					msg.X < x+22 {

					m.modeCompressors()

					return m, nil
				}
			}
		}

		m.stopEditing()

		return m, nil
	}

	if mode == "calculator-only" ||
		width < 80 {

		panelTop := 2
		y := panelTop + 2

		if m.cfg.Calculator.Teeth {

			if msg.X >= 2 &&
				msg.X < 26 &&
				msg.Y == y {

				m.modeTeeth()

				return m, nil
			}

			y += 3
		}

		if m.cfg.Calculator.Compressors {

			if msg.X >= 2 &&
				msg.X < 26 &&
				msg.Y == y {

				m.modeCompressors()

				return m, nil
			}
		}

		m.stopEditing()

		return m, nil
	}

	leftWidth,
		_ :=
		m.panelWidths(
			width,
		)

	calculatorLeft :=
		mode ==
			"calculator-left"

	calculatorX :=
		leftWidth + 1

	if calculatorLeft {
		calculatorX = 0
	}

	fieldX :=
		calculatorX + 2

	teethY := 4
	compressorsY := 7

	const fieldWidth = 24

	if m.cfg.Calculator.Teeth &&
		msg.X >= fieldX &&
		msg.X < fieldX+fieldWidth &&
		msg.Y == teethY {

		m.modeTeeth()

		return m, nil
	}

	if m.cfg.Calculator.Compressors &&
		msg.X >= fieldX &&
		msg.X < fieldX+fieldWidth &&
		msg.Y == compressorsY {

		m.modeCompressors()

		return m, nil
	}

	m.stopEditing()

	return m, nil
}

// ─────────────────────────────────────────────────────────────
// View routing
// ─────────────────────────────────────────────────────────────

func (m model) View() tea.View {

	switch m.page {

	case pageHome:
		return m.viewHome()

	case pageSetup:
		return m.viewSetup()

	case pageMain:
		return m.viewMain()

	case pageCrank:
		return m.viewCrank()

	case pageWheel:
		return m.viewWheel()

	case pageDyno:
		return m.viewDyno()
	case pagePiston:
		return m.viewPiston()

	case pageSettings:
		return m.viewSettings()

	case pageUpdates:
		return m.viewUpdates()

	case pageCustomize:
		return m.viewCustomize()

	case pageToolCustomizeSelect:
		return m.viewToolCustomizeSelect()

	case pageAppearance:
		return m.viewAppearance()

	case pageThemes:
		return m.viewThemes()

	case pageFlavors:
		return m.viewCatppuccinFlavors()

	case pageAccents:
		return m.viewAccents()

	case pageTransparency:
		return m.viewTransparency()

	case pageLayout:
		return m.viewLayout()

	case pageReference:
		return m.viewReference()

	case pageCalculator:
		return m.viewCalculator()

	case pageReset:
		return m.viewReset()

	case pageAbout:
		return m.viewAbout()

	default:
		return m.viewMain()
	}
}

// ─────────────────────────────────────────────────────────────
// Menu renderer
// ─────────────────────────────────────────────────────────────

func (m model) renderMenu(
	title string,
	items []string,
	selected int,
	p palette,
) tea.View {

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

	var lines []string

	lines =
		append(
			lines,
			lipgloss.NewStyle().
				Bold(true).
				Foreground(p.accent).
				Render(title),
			"",
		)

	for i, item := range items {

		prefix :=
			"  "

		if i ==
			selected {

			prefix =
				"> "
		}

		style :=
			lipgloss.NewStyle().
				Foreground(p.text)

		if i ==
			selected {

			style =
				style.
					Foreground(p.accent).
					Background(p.surface).
					Bold(true)
		}

		lines =
			append(
				lines,
				style.Render(
					prefix+item,
				),
			)
	}

	lines =
		append(
			lines,
			"",
			lipgloss.NewStyle().
				Foreground(p.muted).
				Render(
					"[↑↓] Select    [ENTER] Choose    [ESC] Back",
				),
		)

	box :=
		lipgloss.NewStyle().
			Width(62).
			Padding(2, 3).
			Background(p.panel).
			Border(
				lipgloss.NormalBorder(),
			).
			BorderForeground(p.border).
			Render(
				strings.Join(
					lines,
					"\n",
				),
			)

	output :=
		lipgloss.Place(
			width,
			height,
			lipgloss.Center,
			lipgloss.Center,
			box,
		)

	view :=
		tea.NewView(
			output,
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

// ─────────────────────────────────────────────────────────────
// Menu views
// ─────────────────────────────────────────────────────────────

func (m model) viewCustomize() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	return m.renderMenu(
		customizeToolName(
			m.customizeTool,
		)+" CUSTOMIZATION",
		m.customizeItems(),
		m.customIndex,
		p,
	)
}

func (m model) viewToolCustomizeSelect() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	return m.renderMenu(
		"SELECT TOOL TO CUSTOMIZE",
		[]string{
			"Gear Calculator",
			"Crank Angle Calculator",
			"Wheel Calculator",
			"Dyno",
			"Piston Length Calculator",
			"Back",
		},
		m.customToolIndex,
		p,
	)
}

func (m model) viewAppearance() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	background :=
		"Theme"

	if m.cfg.Appearance.Background ==
		"transparent" {

		background =
			"Transparent"
	}

	return m.renderMenu(
		"APPEARANCE",
		[]string{
			"Theme: " +
				themeDisplayName(
					m.cfg.Appearance.Theme,
				),

			"Accent: " +
				accentDisplayName(
					m.cfg.Appearance.Accent,
				),

			"Background: " +
				background,

			"Panel Transparency",

			"Back",
		},
		m.appearanceIndex,
		p,
	)
}

func (m model) viewThemes() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	return m.renderMenu(
		"THEME",
		[]string{
			"Catppuccin",
			"Nord",
			"Gruvbox",
			"Solid",
		},
		m.themeIndex,
		p,
	)
}

func (m model) viewCatppuccinFlavors() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	return m.renderMenu(
		"CATPPUCCIN FLAVOUR",
		m.catppuccinFlavorNamesWithSelection(),
		m.catppuccinFlavorIndex,
		p,
	)
}

func (m model) catppuccinFlavorNamesWithSelection() []string {

	names := m.catppuccinFlavorNames()
	keys := m.catppuccinFlavorKeys()
	items := make([]string, len(names))

	for i := range names {
		items[i] =
			names[i] +
				selectedSuffix(
					m.theme == keys[i],
				)
	}

	return items
}

func (m model) viewAccents() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	return m.renderMenu(
		"ACCENT COLOR",
		accentNames,
		m.accentIndex,
		p,
	)
}

func (m model) viewTransparency() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	return m.renderMenu(
		"PANEL TRANSPARENCY",
		[]string{
			"Top Bar: " +
				boolDisplay(
					m.cfg.Appearance.TopBarTransparent,
				),

			"Reference Panel: " +
				boolDisplay(
					m.cfg.Appearance.ReferenceTransparent,
				),

			"Calculator Panel: " +
				boolDisplay(
					m.cfg.Appearance.CalculatorTransparent,
				),

			"Bottom Bar: " +
				boolDisplay(
					m.cfg.Appearance.BottomBarTransparent,
				),

			"Back",
		},
		m.transparencyIndex,
		p,
	)
}

func (m model) viewLayout() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	items :=
		m.layoutItems()

	if m.customizeTool !=
		customizeGear {

		return m.renderMenu(
			customizeToolName(
				m.customizeTool,
			)+" LAYOUT",
			items,
			m.layoutIndex,
			p,
		)
	}

	mode :=
		m.cfg.Layout.Mode

	if mode == "" {
		mode =
			"calculator-right"
	}

	return m.renderMenu(
		"GEAR CALCULATOR LAYOUT",
		[]string{
			"Automatic" +
				selectedSuffix(
					mode ==
						"automatic",
				),

			"Calculator left" +
				selectedSuffix(
					mode ==
						"calculator-left",
				),

			"Calculator right" +
				selectedSuffix(
					mode ==
						"calculator-right",
				),

			"Calculator only" +
				selectedSuffix(
					mode ==
						"calculator-only",
				),

			"Reference only" +
				selectedSuffix(
					mode ==
						"reference-only",
				),

			"Stacked" +
				selectedSuffix(
					mode ==
						"stacked",
				),

			"Reference width: " +
				layoutWidthDisplay(
					m.cfg.Layout.ReferenceWidth,
				),

			"Stacked order: " +
				orderDisplay(
					m.cfg.Layout.Order,
				),

			"Back",
		},
		m.layoutIndex,
		p,
	)
}

func (m model) viewReference() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	items :=
		[]string{
			"Enabled: " +
				boolDisplay(
					m.cfg.Reference.Enabled,
				),

			"Smallest Gear: " +
				func() string {

					if m.referenceSmallestInput.Focused() {
						return m.referenceSmallestInput.View()
					}

					return strconv.Itoa(
						m.cfg.Reference.SmallestGear,
					)
				}(),

			"Maximum Gear: " +
				func() string {

					if m.referenceMaximumInput.Focused() {
						return m.referenceMaximumInput.View()
					}

					return strconv.Itoa(
						m.cfg.Reference.MaximumGear,
					)
				}(),

			"Teeth: " +
				boolDisplay(
					m.cfg.Reference.Teeth,
				),

			"Full Angle: " +
				boolDisplay(
					m.cfg.Reference.FullAngle,
				),

			"Half Angle: " +
				boolDisplay(
					m.cfg.Reference.HalfAngle,
				),

			"Offset: " +
				boolDisplay(
					m.cfg.Reference.Offset,
				),

			"Compressor Value: " +
				boolDisplay(
					m.cfg.Reference.CompressorValue,
				),

			"Compressors: " +
				boolDisplay(
					m.cfg.Reference.Compressors,
				),

			"Back",
		}

	return m.renderReferenceMenu(
		"REFERENCE CHART",
		items,
		m.referenceIndex,
		p,
	)
}

func (m model) renderReferenceMenu(
	title string,
	items []string,
	selected int,
	p palette,
) tea.View {

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

	boxHeight :=
		20

	if boxHeight >
		height-2 {

		boxHeight =
			height - 2
	}

	if boxHeight < 1 {
		boxHeight = 1
	}

	var lines []string

	titleStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	textStyle :=
		lipgloss.NewStyle().
			Foreground(p.text)

	selectedStyle :=
		lipgloss.NewStyle().
			Foreground(p.accent).
			Background(p.surface).
			Bold(true)

	lines =
		append(
			lines,
			titleStyle.Render(title),
			"",
		)

	for i, item := range items {

		prefix :=
			"  "

		style :=
			textStyle

		if i == selected {

			prefix =
				"> "

			style =
				selectedStyle
		}

		lines =
			append(
				lines,
				style.Render(
					prefix+item,
				),
			)
	}

	lines =
		append(
			lines,
			"",
			lipgloss.NewStyle().
				Foreground(p.muted).
				Render(
					"[↑↓] Select    [ENTER] Edit    [ESC] Back",
				),
		)

	boxWidth :=
		76

	if boxWidth >
		width-4 {

		boxWidth =
			width - 4
	}

	if boxWidth < 20 {
		boxWidth = width
	}

	box :=
		lipgloss.NewStyle().
			Width(boxWidth).
			Height(boxHeight).
			Padding(2, 3).
			Background(p.panel).
			Border(
				lipgloss.NormalBorder(),
			).
			BorderForeground(p.border).
			Render(
				strings.Join(
					lines,
					"\n",
				),
			)

	output :=
		lipgloss.Place(
			width,
			height,
			lipgloss.Center,
			lipgloss.Center,
			box,
		)

	view :=
		tea.NewView(
			output,
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

func (m model) viewCalculator() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	return m.renderMenu(
		"CALCULATOR",
		[]string{
			"Enabled: " +
				boolDisplay(
					m.cfg.Calculator.Enabled,
				),

			"Number of Teeth: " +
				boolDisplay(
					m.cfg.Calculator.Teeth,
				),

			"Compressors: " +
				boolDisplay(
					m.cfg.Calculator.Compressors,
				),

			"Full Angle: " +
				boolDisplay(
					m.cfg.Calculator.FullAngle,
				),

			"Half Angle: " +
				boolDisplay(
					m.cfg.Calculator.HalfAngle,
				),

			"Offset: " +
				boolDisplay(
					m.cfg.Calculator.Offset,
				),

			"Compressor Value: " +
				boolDisplay(
					m.cfg.Calculator.CompressorValue,
				),

			"Warnings: " +
				boolDisplay(
					m.cfg.Calculator.Warnings,
				),

			"Back",
		},
		m.calculatorIndex,
		p,
	)
}

func (m model) viewReset() tea.View {

	p :=
		getPalette(
			m.theme,
			m.accent,
		)

	return m.renderMenu(
		"RESET TO DEFAULTS",
		[]string{
			"YES - Reset everything",
			"NO - Cancel",
		},
		m.resetIndex,
		p,
	)
}

// ─────────────────────────────────────────────────────────────
// About
// ─────────────────────────────────────────────────────────────

func (m model) viewAbout() tea.View {

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

	titleStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	textStyle :=
		lipgloss.NewStyle().
			Foreground(p.text)

	mutedStyle :=
		lipgloss.NewStyle().
			Foreground(p.muted)

	content :=
		strings.Join(
			[]string{
				titleStyle.Render(
					"PC MULTITOOL",
				),
				"",
				textStyle.Render(
					"Plane Crazy Engineering Tools",
				),
				"",
				textStyle.Render(
					"Made by Xad0",
				),
				"",
				textStyle.Render(
					"Version " + Version,
				),
				"",
				mutedStyle.Render(
					"[ESC] Back",
				),
			},
			"\n",
		)

	boxWidth := 42

	if width-8 < boxWidth {
		boxWidth = width - 8
	}

	if boxWidth < 20 {
		boxWidth = width
	}

	box :=
		lipgloss.NewStyle().
			Width(boxWidth).
			Padding(2, 4).
			Foreground(p.text)

	if m.cfg.Appearance.Background !=
		"transparent" {

		box =
			box.Background(
				p.panel,
			)
	}

	rendered :=
		box.Render(content)

	output :=
		lipgloss.Place(
			width,
			height,
			lipgloss.Center,
			lipgloss.Center,
			rendered,
		)

	view :=
		tea.NewView(
			output,
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

// ─────────────────────────────────────────────────────────────
// Reference content builder
// ─────────────────────────────────────────────────────────────

func (m model) referenceContent(
	panelHeight int,
	panelWidth int,
	p palette,
) string {

	sectionStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	var lines []string

	lines =
		append(
			lines,
			sectionStyle.Render(
				fmt.Sprintf(
					"REFERENCE / %d–%d TEETH",
					m.cfg.Reference.SmallestGear,
					m.cfg.Reference.MaximumGear,
				),
			),
			"",
		)

	columns :=
		referenceColumns(
			m.cfg.Reference,
		)

	if len(columns) == 0 {
		return strings.Join(
			lines,
			"\n",
		)
	}

	spacing := 4

	for len(columns) > 1 {
		requiredWidth :=
			referenceColumnsWidth(
				columns,
				spacing,
			)

		if requiredWidth <= panelWidth-4 ||
			spacing == 0 {
			break
		}

		spacing--
	}

	lines =
		append(
			lines,
			buildReferenceHeader(
				columns,
				spacing,
			),
			buildReferenceDivider(
				columns,
				spacing,
			),
			"",
		)

	minGear :=
		m.cfg.Reference.SmallestGear

	maxGear :=
		m.cfg.Reference.MaximumGear

	totalRows :=
		maxGear -
			minGear +
			1

	if totalRows < 1 {
		totalRows = 1
	}

	visibleRows :=
		panelHeight - 5

	if visibleRows < 1 {
		visibleRows = 1
	}

	needsScroll :=
		totalRows >
			visibleRows

	if needsScroll {
		visibleRows--

		if visibleRows < 1 {
			visibleRows = 1
		}
	}

	maxScroll :=
		totalRows -
			visibleRows

	if maxScroll < 0 {
		maxScroll = 0
	}

	scroll :=
		m.referenceScroll

	if scroll < 0 {
		scroll = 0
	}

	if scroll > maxScroll {
		scroll = maxScroll
	}

	startGear :=
		minGear +
			scroll

	endGear :=
		startGear +
			visibleRows -
			1

	if endGear > maxGear {
		endGear =
			maxGear
	}

	for n :=
		startGear; n <= endGear; n++ {

		f,
			h,
			o :=
			Calculate(n)

		c :=
			AutoCompressors(o)

		cv :=
			Round3(
				CompressorValue(
					o,
					c,
				),
			)

		lines =
			append(
				lines,
				buildReferenceRow(
					n,
					f,
					h,
					o,
					cv,
					c,
					columns,
					spacing,
				),
			)
	}

	if needsScroll {

		indicator :=
			""

		if scroll > 0 {
			indicator =
				"↑ more gears above"
		}

		if scroll < maxScroll {

			if indicator != "" {
				indicator +=
					"    "
			}

			indicator +=
				"↓ more gears below"
		}

		if indicator != "" {
			lines =
				append(
					lines,
					indicator,
				)
		}
	}

	// Hard vertical bound.
	if len(lines) > panelHeight {
		lines =
			lines[:panelHeight]
	}

	return strings.Join(
		lines,
		"\n",
	)
}

// ─────────────────────────────────────────────────────────────
// Main rendering
// ─────────────────────────────────────────────────────────────

func (m model) viewMain() tea.View {

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

	teeth :=
		m.currentTeeth()

	compressors :=
		m.currentCompressors()

	full,
		half,
		offset :=
		Calculate(teeth)

	compressorValue :=
		CompressorValue(
			offset,
			compressors,
		)

	roundedValue :=
		Round3(
			compressorValue,
		)

	sectionStyle :=
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

	fieldStyle :=
		lipgloss.NewStyle().
			Width(20).
			Foreground(p.text).
			Padding(0, 1)

	if !m.cfg.Appearance.CalculatorTransparent &&
		m.cfg.Appearance.Background !=
			"transparent" {

		fieldStyle =
			fieldStyle.Background(
				p.surface,
			)
	}

	focusedFieldStyle :=
		fieldStyle.
			Border(
				lipgloss.NormalBorder(),
			).
			BorderForeground(p.accent)

	titleText :=
		"PC GEAR CALCULATOR"

	authorText :=
		"Made by Xad0"

	titleStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	authorStyle :=
		lipgloss.NewStyle().
			Foreground(p.muted)

	if !m.cfg.Appearance.TopBarTransparent &&
		m.cfg.Appearance.Background !=
			"transparent" {

		titleStyle =
			titleStyle.Background(
				p.surface,
			)

		authorStyle =
			authorStyle.Background(
				p.surface,
			)
	}

	title :=
		titleStyle.Render(
			titleText,
		)

	author :=
		authorStyle.Render(
			authorText,
		)

	gap :=
		width -
			lipgloss.Width(titleText) -
			lipgloss.Width(authorText)

	if gap < 1 {
		gap = 1
	}

	gapStyle :=
		lipgloss.NewStyle()

	if !m.cfg.Appearance.TopBarTransparent &&
		m.cfg.Appearance.Background !=
			"transparent" {

		gapStyle =
			gapStyle.Background(
				p.surface,
			)
	}

	topContent :=
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			title,
			gapStyle.Render(
				strings.Repeat(
					" ",
					gap,
				),
			),
			author,
		)

	topRowStyle :=
		lipgloss.NewStyle().
			Width(width).
			Height(1)

	if !m.cfg.Appearance.TopBarTransparent &&
		m.cfg.Appearance.Background !=
			"transparent" {

		topRowStyle =
			topRowStyle.Background(
				p.surface,
			)
	}

	topRow :=
		topRowStyle.Render(
			topContent,
		)

	topSecondStyle :=
		lipgloss.NewStyle().
			Width(width).
			Height(1)

	if !m.cfg.Appearance.TopBarTransparent &&
		m.cfg.Appearance.Background !=
			"transparent" {

		topSecondStyle =
			topSecondStyle.Background(
				p.surface,
			)
	}

	topSecond :=
		topSecondStyle.Render(
			strings.Repeat(
				" ",
				width,
			),
		)

	topBar :=
		lipgloss.JoinVertical(
			lipgloss.Left,
			topRow,
			topSecond,
		)

	contentHeight :=
		height - 4

	if contentHeight < 1 {
		contentHeight = 1
	}

	mode :=
		m.effectiveLayout()

	var reference string

	if m.cfg.Reference.Enabled {

		referencePanelHeight :=
			contentHeight

		if mode ==
			"stacked" {

			calculatorHeight :=
				5

			if contentHeight < 8 {
				calculatorHeight =
					contentHeight / 2
			}

			if calculatorHeight < 1 {
				calculatorHeight = 1
			}

			referencePanelHeight =
				contentHeight -
					calculatorHeight

			if referencePanelHeight < 1 {
				referencePanelHeight = 1
			}
		}

		referenceWidth, _ :=
			m.panelWidths(
				width,
			)

		if mode == "calculator-left" {
			_, referenceWidth =
				m.panelWidths(
					width,
				)
		}

		reference =
			m.referenceContent(
				referencePanelHeight,
				referenceWidth,
				p,
			)
	}

	var calculatorLines []string

	if m.cfg.Calculator.Enabled {

		calculatorLines =
			append(
				calculatorLines,
				sectionStyle.Render(
					"CALCULATOR",
				),
			)

		addGap :=
			func() {

				calculatorLines =
					append(
						calculatorLines,
						"",
					)
			}

		if m.cfg.Calculator.Teeth {

			field :=
				fieldStyle.Render(
					m.teethInput.Value(),
				)

			if m.teethInput.Focused() {

				field =
					focusedFieldStyle.Render(
						m.teethInput.View(),
					)
			}

			calculatorLines =
				append(
					calculatorLines,
					labelStyle.Render(
						"NUMBER OF TEETH",
					),
					field,
				)

			addGap()
		}

		if m.cfg.Calculator.Compressors {

			field :=
				fieldStyle.Render(
					m.compressorInput.Value(),
				)

			if m.compressorInput.Focused() {

				field =
					focusedFieldStyle.Render(
						m.compressorInput.View(),
					)
			}

			calculatorLines =
				append(
					calculatorLines,
					labelStyle.Render(
						"COMPRESSORS",
					),
					field,
				)

			addGap()
		}

		appendValue :=
			func(
				label string,
				value string,
			) {

				calculatorLines =
					append(
						calculatorLines,
						labelStyle.Render(
							label,
						),
						valueStyle.Render(
							value,
						),
					)

				addGap()
			}

		if m.cfg.Calculator.FullAngle {

			appendValue(
				"FULL ANGLE",
				fmt.Sprintf(
					"%.3f°",
					full,
				),
			)
		}

		if m.cfg.Calculator.HalfAngle {

			appendValue(
				"HALF ANGLE",
				fmt.Sprintf(
					"%.3f°",
					half,
				),
			)
		}

		if m.cfg.Calculator.Offset {

			appendValue(
				"OFFSET",
				fmt.Sprintf(
					"%.3f",
					offset,
				),
			)
		}

		if m.cfg.Calculator.CompressorValue {

			appendValue(
				"COMPRESSOR VALUE",
				fmt.Sprintf(
					"%.3f",
					roundedValue,
				),
			)
		}

		if m.cfg.Calculator.Warnings {

			switch {

			case compressorValue > 1:

				calculatorLines =
					append(
						calculatorLines,
						lipgloss.NewStyle().
							Foreground(
								lipgloss.Color(
									"#f9e2af",
								),
							).
							Bold(true).
							Render(
								"MORE COMPRESSORS NEEDED",
							),
					)
			}
		}
	}

	calculator :=
		strings.Join(
			calculatorLines,
			"\n",
		)

	if !m.cfg.Reference.Enabled &&
		!m.cfg.Calculator.Enabled {

		message :=
			lipgloss.NewStyle().
				Foreground(p.muted).
				Bold(true).
				Render(
					"Both the reference chart and calculator are disabled.\n\nPress C to open customization.",
				)

		return m.renderSinglePanel(
			topBar,
			message,
			width,
			height,
			contentHeight,
			p,
			false,
		)
	}

	if mode ==
		"calculator-only" ||
		!m.cfg.Reference.Enabled {

		return m.renderSinglePanel(
			topBar,
			calculator,
			width,
			height,
			contentHeight,
			p,
			m.cfg.Appearance.CalculatorTransparent,
		)
	}

	if mode ==
		"reference-only" ||
		!m.cfg.Calculator.Enabled {

		return m.renderSinglePanel(
			topBar,
			reference,
			width,
			height,
			contentHeight,
			p,
			m.cfg.Appearance.ReferenceTransparent,
		)
	}

	if mode == "stacked" {

		calculatorHeight :=
			5

		if contentHeight < 8 {
			calculatorHeight =
				contentHeight / 2
		}

		if calculatorHeight < 1 {
			calculatorHeight = 1
		}

		referenceHeight :=
			contentHeight -
				calculatorHeight

		if referenceHeight < 1 {
			referenceHeight = 1
		}

		referencePanel :=
			m.renderPanel(
				reference,
				width,
				referenceHeight,
				p,
				m.cfg.Appearance.ReferenceTransparent,
			)

		stackedCalculator :=
			m.renderStackedCalculator(
				width,
				calculatorHeight,
				p,
				full,
				half,
				offset,
				compressorValue,
				roundedValue,
				fieldStyle,
				focusedFieldStyle,
				labelStyle,
				valueStyle,
			)

		var main string

		if m.cfg.Layout.Order ==
			"calculator-first" {

			main =
				lipgloss.JoinVertical(
					lipgloss.Left,
					stackedCalculator,
					referencePanel,
				)

		} else {

			main =
				lipgloss.JoinVertical(
					lipgloss.Left,
					referencePanel,
					stackedCalculator,
				)
		}

		return m.renderMainWithBottom(
			topBar,
			main,
			width,
			p,
		)
	}

	leftWidth,
		rightWidth :=
		m.panelWidths(
			width,
		)

	leftText :=
		reference

	rightText :=
		calculator

	leftTransparent :=
		m.cfg.Appearance.ReferenceTransparent

	rightTransparent :=
		m.cfg.Appearance.CalculatorTransparent

	if mode ==
		"calculator-left" {

		leftText =
			calculator

		rightText =
			reference

		leftTransparent =
			m.cfg.Appearance.CalculatorTransparent

		rightTransparent =
			m.cfg.Appearance.ReferenceTransparent

		leftWidth,
			rightWidth =
			rightWidth,
			leftWidth
	}

	leftPanel :=
		m.renderPanel(
			leftText,
			leftWidth,
			contentHeight,
			p,
			leftTransparent,
		)

	rightPanel :=
		m.renderPanel(
			rightText,
			rightWidth,
			contentHeight,
			p,
			rightTransparent,
		)

	dividerLines :=
		make(
			[]string,
			contentHeight,
		)

	for i := range dividerLines {

		dividerLines[i] =
			"│"
	}

	divider :=
		lipgloss.NewStyle().
			Width(1).
			Height(contentHeight).
			Foreground(p.border).
			Render(
				strings.Join(
					dividerLines,
					"\n",
				),
			)

	main :=
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftPanel,
			divider,
			rightPanel,
		)

	return m.renderMainWithBottom(
		topBar,
		main,
		width,
		p,
	)
}

// ─────────────────────────────────────────────────────────────
// Horizontal stacked calculator
// ─────────────────────────────────────────────────────────────

func (m model) renderStackedCalculator(
	width int,
	height int,
	p palette,
	full float64,
	half float64,
	offset float64,
	compressorValue float64,
	roundedValue float64,
	fieldStyle lipgloss.Style,
	focusedFieldStyle lipgloss.Style,
	labelStyle lipgloss.Style,
	valueStyle lipgloss.Style,
) string {

	title :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent).
			Render(
				"CALCULATOR",
			)

	type calcItem struct {
		label string
		value string
		input bool
		kind  string
	}

	var items []calcItem

	if m.cfg.Calculator.Teeth {

		items =
			append(
				items,
				calcItem{
					label: "TEETH",
					input: true,
					kind:  "teeth",
				},
			)
	}

	if m.cfg.Calculator.Compressors {

		items =
			append(
				items,
				calcItem{
					label: "COMPRESSORS",
					input: true,
					kind:  "compressors",
				},
			)
	}

	if m.cfg.Calculator.FullAngle {

		items =
			append(
				items,
				calcItem{
					label: "FULL",
					value: fmt.Sprintf(
						"%.3f°",
						full,
					),
				},
			)
	}

	if m.cfg.Calculator.HalfAngle {

		items =
			append(
				items,
				calcItem{
					label: "HALF",
					value: fmt.Sprintf(
						"%.3f°",
						half,
					),
				},
			)
	}

	if m.cfg.Calculator.Offset {

		items =
			append(
				items,
				calcItem{
					label: "OFFSET",
					value: fmt.Sprintf(
						"%.3f",
						offset,
					),
				},
			)
	}

	if m.cfg.Calculator.CompressorValue {

		items =
			append(
				items,
				calcItem{
					label: "COMP",
					value: fmt.Sprintf(
						"%.3f",
						roundedValue,
					),
				},
			)
	}

	if len(items) == 0 {

		content :=
			lipgloss.JoinVertical(
				lipgloss.Left,
				title,
				labelStyle.Render(
					"No calculator elements enabled.",
				),
			)

		return m.renderPanel(
			content,
			width,
			height,
			p,
			m.cfg.Appearance.CalculatorTransparent,
		)
	}

	var blocks []string

	for _, item := range items {

		var value string

		switch item.kind {

		case "teeth":

			if m.teethInput.Focused() {

				value =
					focusedFieldStyle.Render(
						m.teethInput.View(),
					)

			} else {

				value =
					fieldStyle.Render(
						m.teethInput.Value(),
					)
			}

		case "compressors":

			if m.compressorInput.Focused() {

				value =
					focusedFieldStyle.Render(
						m.compressorInput.View(),
					)

			} else {

				value =
					fieldStyle.Render(
						m.compressorInput.Value(),
					)
			}

		default:

			value =
				valueStyle.Render(
					item.value,
				)
		}

		block :=
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().
					Bold(true).
					Foreground(p.muted).
					Render(
						item.label,
					),
				value,
			)

		blocks =
			append(
				blocks,
				block,
			)
	}

	row :=
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			blocks...,
		)

	content :=
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			row,
		)

	if m.cfg.Calculator.Warnings {

		switch {

		case compressorValue > 1:

			content =
				lipgloss.JoinVertical(
					lipgloss.Left,
					content,
					lipgloss.NewStyle().
						Foreground(
							lipgloss.Color(
								"#f9e2af",
							),
						).
						Bold(true).
						Render(
							"MORE COMPRESSORS NEEDED",
						),
				)
		}
	}

	return m.renderPanel(
		content,
		width,
		height,
		p,
		m.cfg.Appearance.CalculatorTransparent,
	)
}

// ─────────────────────────────────────────────────────────────
// Shared panel helpers
// ─────────────────────────────────────────────────────────────

func (m model) renderMainWithBottom(
	topBar string,
	main string,
	width int,
	p palette,
) tea.View {

	bottomBar :=
		m.renderBottomBar(
			width,
			p,
		)

	output :=
		strings.Join(
			[]string{
				topBar,
				main,
				bottomBar,
			},
			"\n",
		)

	view :=
		tea.NewView(
			output,
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

func (m model) renderPanel(
	content string,
	width int,
	height int,
	p palette,
	transparent bool,
) string {

	if width < 1 {
		width = 1
	}

	if height < 1 {
		height = 1
	}

	style :=
		lipgloss.NewStyle().
			Width(width).
			Height(height).
			Padding(0, 2).
			Foreground(p.text)

	if !transparent &&
		m.cfg.Appearance.Background !=
			"transparent" {

		style =
			style.Background(
				p.panel,
			)
	}

	return style.Render(content)
}

func (m model) renderSinglePanel(
	topBar string,
	content string,
	width int,
	height int,
	contentHeight int,
	p palette,
	transparent bool,
) tea.View {

	panel :=
		m.renderPanel(
			content,
			width,
			contentHeight,
			p,
			transparent,
		)

	bottomBar :=
		m.renderBottomBar(
			width,
			p,
		)

	output :=
		strings.Join(
			[]string{
				topBar,
				panel,
				bottomBar,
			},
			"\n",
		)

	view :=
		tea.NewView(
			output,
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

func (m model) renderBottomBar(
	width int,
	p palette,
) string {

	text :=
		"[E] Edit teeth    [R] Edit compressors    [C] Customize    [Q] Quit"

	style :=
		lipgloss.NewStyle().
			Width(width).
			Height(2).
			Foreground(p.muted)

	if !m.cfg.Appearance.BottomBarTransparent &&
		m.cfg.Appearance.Background !=
			"transparent" {

		style =
			style.Background(
				p.surface,
			)
	}

	return style.Render(text)
}

// ─────────────────────────────────────────────────────────────
// Reference table
// ─────────────────────────────────────────────────────────────

type refColumn int

const (
	refTeeth refColumn = iota
	refFull
	refHalf
	refOffset
	refCompValue
	refCompressors
)

func referenceColumns(
	cfg referenceConfig,
) []refColumn {

	var columns []refColumn

	if cfg.Teeth {

		columns =
			append(
				columns,
				refTeeth,
			)
	}

	if cfg.FullAngle {

		columns =
			append(
				columns,
				refFull,
			)
	}

	if cfg.HalfAngle {

		columns =
			append(
				columns,
				refHalf,
			)
	}

	if cfg.Offset {

		columns =
			append(
				columns,
				refOffset,
			)
	}

	if cfg.CompressorValue {

		columns =
			append(
				columns,
				refCompValue,
			)
	}

	if cfg.Compressors {

		columns =
			append(
				columns,
				refCompressors,
			)
	}

	return columns
}

func referenceColumnsWidth(
	columns []refColumn,
	spacing int,
) int {

	width := 0

	for _, column := range columns {

		switch column {

		case refTeeth:
			width += 5

		case refFull:
			width += 10

		case refHalf:
			width += 10

		case refOffset:
			width += 8

		case refCompValue:
			width += 10

		case refCompressors:
			width += 10
		}
	}

	if len(columns) > 1 {
		width +=
			(len(columns) - 1) *
				spacing
	}

	return width
}

func buildReferenceHeader(
	columns []refColumn,
	spacing int,
) string {

	var parts []string

	for _, column := range columns {

		switch column {

		case refTeeth:

			parts =
				append(
					parts,
					"TEETH",
				)

		case refFull:

			parts =
				append(
					parts,
					"FULL ANGLE",
				)

		case refHalf:

			parts =
				append(
					parts,
					"HALF ANGLE",
				)

		case refOffset:

			parts =
				append(
					parts,
					"OFFSET",
				)

		case refCompValue:

			parts =
				append(
					parts,
					"COMP VALUE",
				)

		case refCompressors:

			parts =
				append(
					parts,
					"COMPRESSORS",
				)
		}
	}

	return strings.Join(
		parts,
		strings.Repeat(" ", spacing),
	)
}

func buildReferenceDivider(
	columns []refColumn,
	spacing int,
) string {

	var parts []string

	for _, column := range columns {

		switch column {

		case refTeeth:

			parts =
				append(
					parts,
					"─────",
				)

		case refFull:

			parts =
				append(
					parts,
					"──────────",
				)

		case refHalf:

			parts =
				append(
					parts,
					"──────────",
				)

		case refOffset:

			parts =
				append(
					parts,
					"────────",
				)

		case refCompValue:

			parts =
				append(
					parts,
					"──────────",
				)

		case refCompressors:

			parts =
				append(
					parts,
					"───────────",
				)
		}
	}

	return strings.Join(
		parts,
		strings.Repeat(" ", spacing),
	)
}

func buildReferenceRow(
	teeth int,
	full float64,
	half float64,
	offset float64,
	compValue float64,
	compressors int,
	columns []refColumn,
	spacing int,
) string {

	var parts []string

	for _, column := range columns {

		switch column {

		case refTeeth:

			parts =
				append(
					parts,
					fmt.Sprintf(
						"%5d",
						teeth,
					),
				)

		case refFull:

			parts =
				append(
					parts,
					fmt.Sprintf(
						"%9.3f°",
						full,
					),
				)

		case refHalf:

			parts =
				append(
					parts,
					fmt.Sprintf(
						"%9.3f°",
						half,
					),
				)

		case refOffset:

			parts =
				append(
					parts,
					fmt.Sprintf(
						"%8.3f",
						offset,
					),
				)

		case refCompValue:

			parts =
				append(
					parts,
					fmt.Sprintf(
						"%10.3f",
						compValue,
					),
				)

		case refCompressors:

			parts =
				append(
					parts,
					fmt.Sprintf(
						"%10d",
						compressors,
					),
				)
		}
	}

	return strings.Join(
		parts,
		strings.Repeat(" ", spacing),
	)
}

// ─────────────────────────────────────────────────────────────
// Display helpers
// ─────────────────────────────────────────────────────────────

func boolDisplay(
	value bool,
) string {

	if value {
		return "[ON]"
	}

	return "[OFF]"
}

func selectedSuffix(
	value bool,
) string {

	if value {
		return " [SELECTED]"
	}

	return ""
}

func layoutWidthDisplay(
	value string,
) string {

	switch value {

	case "50":
		return "50%"

	case "60":
		return "60%"

	case "70":
		return "70%"

	case "80":
		return "80%"

	default:
		return "Balanced"
	}
}

func orderDisplay(
	value string,
) string {

	if value ==
		"calculator-first" {

		return "Calculator first"
	}

	return "Reference first"
}

func themeDisplayName(
	key string,
) string {

	switch key {
	case "catppuccin-latte":
		return "Catppuccin Latte"
	case "catppuccin-frappe":
		return "Catppuccin Frappé"
	case "catppuccin-macchiato":
		return "Catppuccin Macchiato"
	case "catppuccin-mocha":
		return "Catppuccin Mocha"
	case "nord":
		return "Nord"
	case "gruvbox":
		return "Gruvbox"
	case "solid":
		return "Solid"
	}

	return "Catppuccin Mocha"
}

func accentDisplayName(
	key string,
) string {

	for i, value := range accentKeys {

		if value ==
			key {

			return accentNames[i]
		}
	}

	return "Green"
}
