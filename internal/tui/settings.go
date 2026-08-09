package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/password-memorizer-tui/internal/config"
	"github.com/alankiri/password-memorizer-tui/internal/engine"
	"github.com/alankiri/password-memorizer-tui/internal/levels"
)

// settingsItem represents a single row inside the level settings list.
// A row is either a level header (header == true) or an editable field.
type settingsItem struct {
	levelIdx int
	field    string
	header   bool
}

// levelField describes one of the advanced fields that is hidden by default
// under a level in the settings editor.
type levelField struct {
	name    string
	display string
	isInt   bool
	min     int
	max     int
	desc    string
}

// levelFields is the ordered list of fields shown when a level is expanded.
var levelFields = []levelField{
	{name: "base_blur_percent", display: "Base blur %", isInt: true, min: 0, max: 100, desc: "Percentage of characters blurred at the start of a level."},
	{name: "blur_step_percent", display: "Blur step %", isInt: true, min: 0, max: 100, desc: "How much blur increases after each failed attempt."},
	{name: "max_blur_percent", display: "Max blur %", isInt: true, min: 0, max: 100, desc: "Maximum blur that can be applied to a password."},
	{name: "hint_reduction_percent", display: "Hint reduction %", isInt: true, min: 0, max: 100, desc: "How much the hint shrinks after each attempt."},
	{name: "required_sessions_to_progress", display: "Required sessions", isInt: true, min: 1, desc: "How many counted training sessions are needed before a trainer advances."},
	{name: "session_interval_hours", display: "Session interval (hours)", isInt: true, min: 0, desc: "Minimum hours between counted training sessions for this level."},
	{name: "typing_validation_mode", display: "Typing validation", isInt: false, desc: "Training strictness. Use 'allow_highlight' or 'strict'."},
	{name: "description", display: "Description", isInt: false, desc: "Short description of the level shown in settings."},
}

// resetSettings puts the settings screen into its initial vault state and
// forgets any level expansion, focus, or in-progress edit.
func (m *Model) resetSettings() {
	m.settingsCat = settingsCategoryVault
	m.resetEnableVault()
	m.settingsTabsFocused = true
	m.settingsFocus = 0
	m.settingsExpanded = make([]bool, len(m.levels))
	m.settingsEditing = false
	m.settingsEditInput.SetValue("")
	m.settingsEditErr = ""
}

// resetEnableVault clears the vault password inputs and chooses the right
// starting step based on whether a vault already exists.
// resetForCategory resets the focused section when the category changes.
func (m *Model) resetForCategory() {
	if m.settingsCat == settingsCategoryVault {
		m.resetEnableVault()
	} else {
		m.settingsFocus = 0
	}
}

func (m *Model) resetEnableVault() {
	m.vaultPw.SetValue("")
	m.vaultConfirm.SetValue("")
	m.vaultPw.EchoMode = textinput.EchoPassword
	m.vaultPw.EchoCharacter = '•'
	m.vaultConfirm.EchoMode = textinput.EchoPassword
	m.vaultConfirm.EchoCharacter = '•'
	m.vaultShowPw = false
	m.vaultChoice = 0
	m.vaultErrMsg = ""
	if m.db.HasVault() {
		m.vaultStep = vaultStepMenu
	} else {
		m.vaultStep = vaultStepWarn
		m.vaultPw.Focus()
		m.vaultConfirm.Blur()
	}
}

// applyVaultEcho toggles the password inputs between masked and visible
// based on m.vaultShowPw.
func (m *Model) applyVaultEcho() {
	if m.vaultShowPw {
		m.vaultPw.EchoMode = textinput.EchoNormal
		m.vaultConfirm.EchoMode = textinput.EchoNormal
	} else {
		m.vaultPw.EchoMode = textinput.EchoPassword
		m.vaultConfirm.EchoMode = textinput.EchoPassword
	}
	m.vaultPw.EchoCharacter = '•'
	m.vaultConfirm.EchoCharacter = '•'
}

// updateSettings routes input for the settings screen. Number keys switch
// categories and return focus to the sidebar. Tab toggles focus between the
// sidebar and the active section. Esc exits when the sidebar is focused and
// is delegated to the active section otherwise.
func (m Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()

	// When editing a level field, all input goes to the textinput.
	if m.settingsEditing && m.settingsCat == settingsCategoryLevels {
		return m.updateLevelEdit(msg)
	}

	if s == "q" {
		m.screen = screenList
		return m, nil
	}

	if s == "esc" {
		if m.settingsTabsFocused {
			m.screen = screenList
			return m, nil
		}
		// Let the active section decide what esc means inside it.
		switch m.settingsCat {
		case settingsCategoryVault:
			return m.updateVaultSettings(msg)
		case settingsCategoryLevels:
			return m.updateLevelsSettings(msg)
		}
		return m, nil
	}

	// Tab always toggles focus, even when a section is active.
	if s == "tab" {
		m.settingsTabsFocused = !m.settingsTabsFocused
		return m, nil
	}

	// Number keys switch category and reset the sidebar focus.
	if s == "1" {
		m.settingsCat = settingsCategoryVault
		m.settingsTabsFocused = true
		m.resetForCategory()
		return m, nil
	}
	if s == "2" {
		m.settingsCat = settingsCategoryLevels
		m.settingsTabsFocused = true
		m.resetForCategory()
		return m, nil
	}

	// When the sidebar is focused, j/k also switch categories.
	if m.settingsTabsFocused {
		if s == "j" || s == "down" || s == "k" || s == "up" {
			m.settingsCat = (m.settingsCat + 1) % 2
			m.resetForCategory()
		}
		return m, nil
	}

	switch m.settingsCat {
	case settingsCategoryVault:
		return m.updateVaultSettings(msg)
	case settingsCategoryLevels:
		return m.updateLevelsSettings(msg)
	}
	return m, nil
}

// updateVaultSettings handles the vault sub-states: enable, change, decrypt.
// Warning screens are single-action: enter confirms, esc goes back.
// Password inputs use esc to return to the previous step.
func (m Model) updateVaultSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()

	if s == "esc" {
		switch m.vaultStep {
		case vaultStepWarn, vaultStepMenu:
			m.screen = screenList
		case vaultStepPassword:
			m.vaultStep = vaultStepWarn
		case vaultStepConfirm:
			m.vaultStep = vaultStepPassword
		case vaultStepChangePassword:
			m.vaultStep = vaultStepMenu
		case vaultStepChangeConfirm:
			m.vaultStep = vaultStepChangePassword
		case vaultStepDecryptWarn:
			m.vaultStep = vaultStepMenu
		}
		m.vaultErrMsg = ""
		return m, nil
	}

	if s == "ctrl+s" {
		m.vaultShowPw = !m.vaultShowPw
		m.applyVaultEcho()
		return m, nil
	}
	switch m.vaultStep {
	case vaultStepWarn:
		if s == "enter" {
			m.vaultStep = vaultStepPassword
			m.vaultPw.Focus()
			m.vaultErrMsg = ""
		}
		return m, nil
	case vaultStepPassword:
		if s == "enter" {
			m.vaultStep = vaultStepConfirm
			m.vaultConfirm.Focus()
			m.vaultErrMsg = ""
		}
	case vaultStepConfirm:
		if s == "enter" {
			m.vaultErrMsg = ""
			if m.vaultPw.Value() != m.vaultConfirm.Value() {
				m.vaultErrMsg = "Passwords do not match"
				return m, nil
			}
			v, err := m.db.CreateVaultAndEncrypt(m.vaultPw.Value())
			if err != nil {
				m.vaultErrMsg = err.Error()
				return m, nil
			}
			m.db.SetVault(v)
			m.cfg.PromptedForVault = true
			if err := config.Save(m.cfg); err != nil {
				m.vaultErrMsg = err.Error()
				return m, nil
			}
			m.loadTrainers()
			m.screen = screenList
		}
	case vaultStepMenu:
		if s == "j" || s == "down" {
			m.vaultChoice = 1
			return m, nil
		}
		if s == "k" || s == "up" {
			m.vaultChoice = 0
			return m, nil
		}
		if s == "enter" {
			if m.vaultChoice == 0 {
				m.vaultStep = vaultStepChangePassword
				m.vaultPw.SetValue("")
				m.vaultConfirm.SetValue("")
				m.vaultPw.Focus()
				m.vaultErrMsg = ""
			} else {
				m.vaultStep = vaultStepDecryptWarn
				m.vaultChoice = 0
				m.vaultErrMsg = ""
			}
		}
		return m, nil
	case vaultStepChangePassword:
		if s == "enter" {
			m.vaultStep = vaultStepChangeConfirm
			m.vaultConfirm.Focus()
			m.vaultErrMsg = ""
		}
	case vaultStepChangeConfirm:
		if s == "enter" {
			m.vaultErrMsg = ""
			if m.vaultPw.Value() != m.vaultConfirm.Value() {
				m.vaultErrMsg = "Passwords do not match"
				return m, nil
			}
			v, err := m.db.ChangeVault(m.vaultPw.Value())
			if err != nil {
				m.vaultErrMsg = err.Error()
				return m, nil
			}
			m.db.SetVault(v)
			m.loadTrainers()
			m.screen = screenList
		}
	case vaultStepDecryptWarn:
		if s == "enter" {
			if err := m.db.DecryptVault(); err != nil {
				m.vaultErrMsg = err.Error()
				return m, nil
			}
			m.loadTrainers()
			m.screen = screenList
		}
		return m, nil
	}
	var cmd tea.Cmd
	if m.vaultStep == vaultStepConfirm || m.vaultStep == vaultStepChangeConfirm {
		m.vaultConfirm, cmd = m.vaultConfirm.Update(msg)
	} else if m.vaultStep == vaultStepPassword || m.vaultStep == vaultStepChangePassword {
		m.vaultPw, cmd = m.vaultPw.Update(msg)
	}
	return m, cmd
}

// settingsLevelItems builds a flattened list of level headers and expanded
// fields that the user can navigate with j/k.
func (m *Model) settingsLevelItems() []settingsItem {
	var items []settingsItem
	for i := range m.levels {
		items = append(items, settingsItem{levelIdx: i, header: true})
		if m.settingsExpanded[i] {
			for _, f := range levelFields {
				items = append(items, settingsItem{levelIdx: i, field: f.name})
			}
		}
	}
	return items
}

// updateLevelsSettings handles j/k navigation, expand/collapse (l/h), and
// entering edit mode for a level field.
func (m *Model) updateLevelsSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	items := m.settingsLevelItems()
	if len(items) == 0 {
		return m, nil
	}

	if s == "esc" {
		m.settingsTabsFocused = true
		return m, nil
	}

	if s == "j" || s == "down" {
		if m.settingsFocus < len(items)-1 {
			m.settingsFocus++
		}
		return m, nil
	}
	if s == "k" || s == "up" {
		if m.settingsFocus > 0 {
			m.settingsFocus--
		}
		return m, nil
	}
	if s == "g" {
		m.settingsFocus = 0
		return m, nil
	}
	if s == "G" {
		m.settingsFocus = len(items) - 1
		return m, nil
	}

	// h collapses the focused level if it is expanded and moves focus back
	// to the level header. Works from any expanded item.
	if s == "h" {
		item := items[m.settingsFocus]
		if m.settingsExpanded[item.levelIdx] {
			m.settingsExpanded[item.levelIdx] = false
			for i, it := range items {
				if it.header && it.levelIdx == item.levelIdx {
					m.settingsFocus = i
					break
				}
			}
		}
		return m, nil
	}

	// l or enter expands a level header or starts editing a field.
	if s == "l" || s == "enter" {
		item := items[m.settingsFocus]
		if item.header {
			if !m.settingsExpanded[item.levelIdx] {
				m.settingsExpanded[item.levelIdx] = true
				m.settingsFocus++
			}
			return m, nil
		}
		m.startLevelEdit(item.levelIdx, item.field)
		return m, nil
	}

	return m, nil
}

// startLevelEdit puts the selected level field into edit mode and pre-fills
// the textinput with the current value.
func (m *Model) startLevelEdit(levelIdx int, field string) {
	m.settingsEditing = true
	m.settingsEditIdx = levelIdx
	m.settingsEditField = field
	m.settingsEditErr = ""
	m.settingsEditInput.Focus()
	val := m.getLevelFieldValue(levelIdx, field)
	m.settingsEditInput.SetValue(val)
	m.settingsEditInput.SetCursor(len(val))
}

// getLevelFieldValue returns the current string value of a level field.
func (m Model) getLevelFieldValue(levelIdx int, field string) string {
	lvl := m.levels[levelIdx]
	switch field {
	case "base_blur_percent":
		return strconv.Itoa(lvl.BaseBlurPercent)
	case "blur_step_percent":
		return strconv.Itoa(lvl.BlurStepPercent)
	case "max_blur_percent":
		return strconv.Itoa(lvl.MaxBlurPercent)
	case "hint_reduction_percent":
		return strconv.Itoa(lvl.HintReductionPercent)
	case "required_sessions_to_progress":
		return strconv.Itoa(lvl.RequiredSessionsToProgress)
	case "session_interval_hours":
		return strconv.Itoa(lvl.SessionIntervalHours)
	case "typing_validation_mode":
		return lvl.TypingValidationMode
	case "description":
		return lvl.Description
	}
	return ""
}

// setLevelFieldValue parses and validates a new value for a level field.
func (m *Model) setLevelFieldValue(levelIdx int, field, value string) error {
	lvl := &m.levels[levelIdx]
	switch field {
	case "typing_validation_mode":
		if value == "" {
			return fmt.Errorf("typing validation mode cannot be empty")
		}
		lvl.TypingValidationMode = value
	case "description":
		lvl.Description = value
	default:
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s must be an integer", field)
		}
		for _, f := range levelFields {
			if f.name == field {
				if f.max > 0 && n > f.max {
					return fmt.Errorf("%s must be at most %d", f.display, f.max)
				}
				if n < f.min {
					return fmt.Errorf("%s must be at least %d", f.display, f.min)
				}
				break
			}
		}
		switch field {
		case "base_blur_percent":
			lvl.BaseBlurPercent = n
		case "blur_step_percent":
			lvl.BlurStepPercent = n
		case "max_blur_percent":
			lvl.MaxBlurPercent = n
		case "hint_reduction_percent":
			lvl.HintReductionPercent = n
		case "required_sessions_to_progress":
			lvl.RequiredSessionsToProgress = n
		case "session_interval_hours":
			lvl.SessionIntervalHours = n
		}
	}
	return nil
}

// validateAndSaveLevel applies the edited value, writes the levels yaml, and
// rebuilds the engine so the changes are active immediately.
func (m *Model) validateAndSaveLevel() error {
	val := m.settingsEditInput.Value()
	if err := m.setLevelFieldValue(m.settingsEditIdx, m.settingsEditField, val); err != nil {
		return err
	}
	if err := levels.Save(m.levels); err != nil {
		return err
	}
	m.eng = engine.New(m.levels)
	return nil
}

// updateLevelEdit reads keys while a level field is being edited.
// enter saves the value, esc cancels.
func (m Model) updateLevelEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if s == "esc" {
		m.settingsEditing = false
		m.settingsEditInput.SetValue("")
		m.settingsEditErr = ""
		return m, nil
	}
	if s == "enter" {
		if err := m.validateAndSaveLevel(); err != nil {
			m.settingsEditErr = err.Error()
			return m, nil
		}
		m.settingsEditing = false
		m.settingsEditInput.SetValue("")
		m.settingsEditErr = ""
		return m, nil
	}
	var cmd tea.Cmd
	m.settingsEditInput, cmd = m.settingsEditInput.Update(msg)
	return m, cmd
}

// settingsView renders the full settings screen: title, category sidebar,
// and the active sub-view.
func (m Model) settingsView() string {
	var body strings.Builder
	title := renderTitle("Settings")
	body.WriteString(title)
	body.WriteString("\n\n")

	sidebar := m.settingsSidebar()
	var right string
	switch m.settingsCat {
	case settingsCategoryVault:
		right = m.vaultSettingsView()
	case settingsCategoryLevels:
		right = m.levelsSettingsView()
	}
	body.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sidebar, "  ", right))

	content := body.String()
	contentLines := strings.Count(content, "\n") + 1
	footer := dimStyle().Render("tab: switch focus • 1: vault • 2: levels • esc: back")
	footerLines := strings.Count(footer, "\n") + 1
	if m.height > 0 {
		gap := m.height - contentLines - footerLines
		if gap < 0 {
			gap = 0
		}
		return content + strings.Repeat("\n", gap) + footer
	}
	return content + "\n\n" + footer
}

// settingsSidebar draws the category selection. The active category is
// highlighted when focus is on the sidebar and cream when focus is in the
// section so the focus state is always clear.
func (m Model) settingsSidebar() string {
	var sb strings.Builder
	items := []struct {
		key  string
		name string
	}{
		{"1", "Vault"},
		{"2", "Levels"},
	}
	for i, it := range items {
		isActive := settingsCategory(i) == m.settingsCat
		prefix := "  "
		style := dimStyle()
		if isActive && m.settingsTabsFocused {
			prefix = "> "
			style = lipgloss.NewStyle().Bold(true).Foreground(glow.Fuchsia)
		} else if isActive {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(glow.Cream)
		}
		sb.WriteString(style.Render(prefix + fmt.Sprintf("%s. %s", it.key, it.name)))
		sb.WriteString("\n")
	}
	return lipgloss.NewStyle().Padding(0, 1, 0, 0).Render(sb.String())
}

// vaultSettingsView renders the active vault sub-state. Yes/no prompts are
// presented as a selectable list.
func (m Model) vaultSettingsView() string {
	var body strings.Builder

	switch m.vaultStep {
	case vaultStepWarn:
		body.WriteString(lipgloss.NewStyle().Foreground(glow.Red).Bold(true).Render("WARNING: this will encrypt all stored trainer passwords."))
		body.WriteString("\n\n")
		body.WriteString("Once enabled, you will need the master password every time you start passmem.\n")
		body.WriteString("If you forget it, there is no way to recover your stored passwords.\n\n")
		body.WriteString(m.vaultActionButton("Enable vault"))
	case vaultStepPassword:
		body.WriteString("Create a master password for your vault.\n\n")
		body.WriteString(m.vaultPw.View())
	case vaultStepConfirm:
		body.WriteString("Re-enter the master password to confirm.\n\n")
		body.WriteString(m.vaultConfirm.View())
	case vaultStepMenu:
		body.WriteString("Vault is enabled.\n\n")
		body.WriteString(m.choiceList([]string{"Change master password", "Decrypt and remove vault"}, m.vaultChoice))
	case vaultStepChangePassword:
		body.WriteString("Enter a new master password.\n\n")
		body.WriteString(m.vaultPw.View())
	case vaultStepChangeConfirm:
		body.WriteString("Re-enter the new master password to confirm.\n\n")
		body.WriteString(m.vaultConfirm.View())
	case vaultStepDecryptWarn:
		body.WriteString(lipgloss.NewStyle().Foreground(glow.Red).Bold(true).Render("WARNING: this will decrypt all stored passwords."))
		body.WriteString("\n\n")
		body.WriteString("The vault will be removed and you will no longer need a master password.\n\n")
		body.WriteString(m.vaultActionButton("Remove vault"))
	}
	body.WriteString("\n")
	if m.vaultErrMsg != "" {
		body.WriteString(lipgloss.NewStyle().Foreground(glow.Red).Render(m.vaultErrMsg))
		body.WriteString("\n")
	}

	var guide string
	switch m.vaultStep {
	case vaultStepWarn, vaultStepDecryptWarn:
		guide = "enter: confirm • esc: back"
	case vaultStepMenu:
		guide = "j/k: choose • enter: confirm • esc: back"
	case vaultStepPassword, vaultStepChangePassword:
		guide = "enter: continue • ctrl+s: show • esc: back"
	case vaultStepConfirm, vaultStepChangeConfirm:
		guide = "enter: confirm • ctrl+s: show • esc: back"
	}
	footer := dimStyle().Render(guide)
	return body.String() + "\n" + footer
}

// choiceList renders a vertical list of options. The arrow is always on the
// last selected item and turns blue when the section is focused.
func (m Model) choiceList(options []string, selected int) string {
	var b strings.Builder
	for i, opt := range options {
		var style lipgloss.Style
		prefix := "  "
		if i == selected {
			prefix = "> "
			style = lipgloss.NewStyle().Bold(true).Foreground(glow.Cream)
			if !m.settingsTabsFocused {
				style = lipgloss.NewStyle().Bold(true).Foreground(glow.Blue)
			}
		} else {
			style = lipgloss.NewStyle().Foreground(glow.Cream)
		}
		b.WriteString(style.Render(prefix + opt))
		b.WriteString("\n")
	}
	return b.String()
}

// vaultActionButton renders the single action on a warning screen. It keeps an
// arrow at all times and is blue only when the settings section has focus.
func (m Model) vaultActionButton(label string) string {
	style := lipgloss.NewStyle().Foreground(glow.Cream)
	if !m.settingsTabsFocused {
		style = lipgloss.NewStyle().Bold(true).Foreground(glow.Blue)
	}
	return style.Render("> " + label)
}

// levelsSettingsView renders the level editor. Each level is collapsed by
// default and expands to show the editable advanced fields. Field
// descriptions for the focused row are shown to the right, similar to the
// level picker when creating a new trainer.
func (m Model) levelsSettingsView() string {
	var body strings.Builder
	items := m.settingsLevelItems()

	for i, item := range items {
		if item.header {
			lvl := m.levels[item.levelIdx]
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(glow.Cream)
			if i == m.settingsFocus {
				prefix = "> "
				if !m.settingsTabsFocused {
					style = lipgloss.NewStyle().Bold(true).Foreground(glow.Blue)
				}
			}
			state := "[expand]"
			if m.settingsExpanded[item.levelIdx] {
				state = "[collapse]"
			}
			line := style.Render(prefix+fmt.Sprintf("Level %d — %s", lvl.Number, lvl.Description)) + " " + dimStyle().Render(state)
			body.WriteString(line)
		} else {
			for _, f := range levelFields {
				if f.name == item.field {
					val := m.getLevelFieldValue(item.levelIdx, item.field)
					focused := i == m.settingsFocus
					body.WriteString(m.renderLevelFieldRow(f, val, focused))
					break
				}
			}
		}
		body.WriteString("\n")
	}

	if len(items) == 0 {
		body.WriteString("No levels configured.")
		body.WriteString("\n")
	}

	if m.settingsEditing {
		body.WriteString("\n")
		body.WriteString(m.settingsEditInput.View())
		body.WriteString("\n")
		if m.settingsEditErr != "" {
			body.WriteString(lipgloss.NewStyle().Foreground(glow.Red).Render(m.settingsEditErr))
			body.WriteString("\n")
		} else {
			body.WriteString(dimStyle().Render("enter: save • esc: cancel"))
			body.WriteString("\n")
		}
	} else {
		body.WriteString("\n")
		body.WriteString(dimStyle().Render("j/k: navigate • enter: expand/edit"))
		body.WriteString("\n")
	}
	return body.String()
}

// renderLevelFieldRow draws one field value. The focused item keeps an arrow
// and is blue only when the level list has focus.
func (m Model) renderLevelFieldRow(f levelField, value string, focused bool) string {
	prefix := "    "
	if focused {
		prefix = ">   "
	}
	leftText := prefix + fmt.Sprintf("%s: %s", f.display, value)
	leftStyle := lipgloss.NewStyle().Foreground(glow.Cream)
	if focused && !m.settingsTabsFocused {
		leftStyle = lipgloss.NewStyle().Foreground(glow.Blue)
	}
	left := leftStyle.Render(leftText)

	if !focused {
		return left
	}

	// Reserve space for the label side so the description lines up.
	leftWidth := 35
	descWidth := m.width - leftWidth - 4
	if descWidth < 20 {
		descWidth = 20
	}
	right := lipgloss.NewStyle().
		Foreground(glow.Dim).
		Width(descWidth).
		Render(f.desc)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}
