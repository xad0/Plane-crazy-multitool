package main

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type setupField int

const (
	setupExport setupField = iota
	setupTheme
	setupAccent
	setupFinish
	setupSkip
)

const setupFieldCount = 5

func (m model) setupItems() []string {
	return []string{
		"Export folder",
		"Theme",
		"Accent",
		"Finish setup",
		"Skip setup",
	}
}

func (m model) setupExportDisplay() string {
	path := m.cfg.ExportDirectory

	if path == "" {
		path = defaultExportDirectory()
	}

	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}

	return path
}

func (m model) updateSetup(
	msg tea.KeyPressMsg,
) (tea.Model, tea.Cmd) {

	// Export path is being edited.
	if m.setupExportInput.Focused() {

		switch msg.String() {

		case "enter":
			value :=
				strings.TrimSpace(
					m.setupExportInput.Value(),
				)

			if value != "" {
				m.cfg.ExportDirectory =
					value
			}

			m.setupExportInput.Blur()

			return m, nil

		case "esc":
			m.setupExportInput.Blur()

			return m, nil
		}

		var cmd tea.Cmd

		m.setupExportInput,
			cmd =
			m.setupExportInput.Update(msg)

		return m, cmd
	}

	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up":

		m.setupIndex--

		if m.setupIndex < 0 {
			m.setupIndex =
				setupFieldCount - 1
		}

	case "down":

		m.setupIndex++

		if m.setupIndex >= setupFieldCount {
			m.setupIndex = 0
		}

	case "left":

		switch setupField(m.setupIndex) {

		case setupTheme:
			m.cycleSetupTheme(-1)

		case setupAccent:
			m.cycleSetupAccent(-1)

		}

	case "right":

		switch setupField(m.setupIndex) {

		case setupTheme:
			m.cycleSetupTheme(1)

		case setupAccent:
			m.cycleSetupAccent(1)

		}

	case "enter":

		switch setupField(m.setupIndex) {

		case setupExport:

			m.setupExportInput.SetValue(
				m.cfg.ExportDirectory,
			)

			m.setupExportInput.Focus()
			m.setupExportInput.CursorEnd()

			return m, nil

		case setupTheme:

			m.cycleSetupTheme(1)

		case setupAccent:

			m.cycleSetupAccent(1)

		case setupFinish:

			m.finishSetup()

			m.page =
				pageHome

			return m, nil

		case setupSkip:

			m.cfg.SetupCompleted =
				true

			m.saveSettings()

			m.page =
				pageHome

			return m, nil
		}

	case "s":

		m.cfg.SetupCompleted =
			true

		m.saveSettings()

		m.page =
			pageHome

		return m, nil

	case "esc":

		m.cfg.SetupCompleted =
			true

		m.saveSettings()

		m.page =
			pageHome

		return m, nil
	}

	return m, nil
}

func (m *model) finishSetup() {

	if m.setupExportInput.Focused() {

		value :=
			strings.TrimSpace(
				m.setupExportInput.Value(),
			)

		if value != "" {
			m.cfg.ExportDirectory =
				value
		}

		m.setupExportInput.Blur()
	}

	if m.cfg.ExportDirectory == "" {
		m.cfg.ExportDirectory =
			defaultExportDirectory()
	}

	m.cfg.SetupCompleted =
		true

	m.saveSettings()
}

func (m *model) cycleSetupTheme(
	delta int,
) {

	current :=
		m.themeIndex

	current += delta

	for current < 0 {
		current += len(themeKeys)
	}

	current %=
		len(themeKeys)

	m.themeIndex =
		current

	m.theme =
		themeKeys[current]

	m.cfg.Appearance.Theme =
		m.theme

	m.saveSettings()
}

func (m *model) cycleSetupAccent(
	delta int,
) {

	current :=
		m.accentIndex

	current += delta

	for current < 0 {
		current += len(accentKeys)
	}

	current %=
		len(accentKeys)

	m.accentIndex =
		current

	m.accent =
		accentKeys[current]

	m.cfg.Appearance.Accent =
		m.accent

	m.saveSettings()
}

func (m model) viewSetup() tea.View {

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

	subtitleStyle :=
		lipgloss.NewStyle().
			Foreground(p.muted)

	labelStyle :=
		lipgloss.NewStyle().
			Foreground(p.text)

	selectedStyle :=
		lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent)

	valueStyle :=
		lipgloss.NewStyle().
			Foreground(p.text)

	var lines []string

	lines =
		append(
			lines,
			titleStyle.Render(
				"PC MULTITOOL SETUP",
			),
			"",
			subtitleStyle.Render(
				"Welcome! Let's configure a few things before you start.",
			),
			"",
		)

	items :=
		m.setupItems()

	for i, item := range items {

		prefix :=
			"  "

		style :=
			labelStyle

		if i ==
			m.setupIndex {

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

		switch setupField(i) {

		case setupExport:

			if m.setupExportInput.Focused() {

				inputStyle :=
					lipgloss.NewStyle().
						Width(64).
						Foreground(p.text).
						Background(p.surface).
						Padding(0, 1)

				lines =
					append(
						lines,
						inputStyle.Render(
							m.setupExportInput.View(),
						),
					)

			} else {

				lines =
					append(
						lines,
						valueStyle.Render(
							"    "+
								m.setupExportDisplay(),
						),
					)
			}

		case setupTheme:

			lines =
				append(
					lines,
					valueStyle.Render(
						"    "+
							themeDisplayName(
								m.cfg.Appearance.Theme,
							),
					),
				)

		case setupAccent:

			lines =
				append(
					lines,
					valueStyle.Render(
						"    "+
							accentDisplayName(
								m.cfg.Appearance.Accent,
							),
					),
				)

		case setupFinish:

			lines =
				append(
					lines,
					valueStyle.Render(
						"    Save settings and continue",
					),
				)

		case setupSkip:

			lines =
				append(
					lines,
					valueStyle.Render(
						"    Use defaults and continue",
					),
				)
		}

		lines =
			append(
				lines,
				"",
			)
	}

	lines =
		append(
			lines,
			subtitleStyle.Render(
				"[↑↓] Select    [←→] Change    [ENTER] Edit",
			),
			subtitleStyle.Render(
				"[S] Skip setup    [ESC] Skip setup",
			),
		)

	content :=
		strings.Join(
			lines,
			"\n",
		)

	box :=
		lipgloss.NewStyle().
			Width(72).
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

	view.AltScreen =
		true

	view.MouseMode =
		tea.MouseModeCellMotion

	if m.cfg.Appearance.Background !=
		"transparent" {

		view.BackgroundColor =
			p.background
	}

	return view
}

func (m model) handleSetupMouse(
	msg tea.MouseClickMsg,
) (tea.Model, tea.Cmd) {

	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	/*
		Setup layout positions:

		6  Export folder
		7  Export path

		8  Theme
		9  Theme value

		10 Accent
		11 Accent value

		12 Finish setup
		13 Description

		14 Skip setup
		15 Description
	*/

	switch msg.Y {

	case 6, 7:

		m.setupIndex =
			int(setupExport)

		return m, nil

	case 8, 9:

		m.setupIndex =
			int(setupTheme)

		return m, nil

	case 10, 11:

		m.setupIndex =
			int(setupAccent)

		return m, nil

	case 12, 13:

		m.setupIndex =
			int(setupFinish)

		m.finishSetup()
		m.page =
			pageHome

		return m, nil

	case 14, 15:

		m.setupIndex =
			int(setupSkip)

		m.cfg.SetupCompleted =
			true

		m.saveSettings()

		m.page =
			pageHome

		return m, nil
	}

	return m, nil
}
