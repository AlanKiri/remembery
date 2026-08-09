package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/password-memorizer-tui/internal/beep"
	"github.com/alankiri/password-memorizer-tui/internal/config"
	"github.com/alankiri/password-memorizer-tui/internal/consts"
	"github.com/alankiri/password-memorizer-tui/internal/engine"
	"github.com/alankiri/password-memorizer-tui/internal/levels"
	"github.com/alankiri/password-memorizer-tui/internal/store"
)

var glow = struct {
	Fuchsia     lipgloss.Color
	DullFuchsia lipgloss.Color
	LightBlue   lipgloss.Color
	Blue        lipgloss.Color
	Green       lipgloss.Color
	Red         lipgloss.Color
	Yellow      lipgloss.Color
	Cream       lipgloss.Color
	Dim         lipgloss.Color
	White       lipgloss.Color
}{
	Fuchsia:     lipgloss.Color("#EE6FF8"),
	DullFuchsia: lipgloss.Color("#F793FF"),
	LightBlue:   lipgloss.Color("#8BE9FD"),
	Blue:        lipgloss.Color("#2A7FFF"),
	Green:       lipgloss.Color("#04B575"),
	Red:         lipgloss.Color("#FF4672"),
	Yellow:      lipgloss.Color("#ECFD65"),
	Cream:       lipgloss.Color("#FFFDF5"),
	Dim:         lipgloss.Color("#777777"),
	White:       lipgloss.Color("#FFFDF5"),
}

type welcomeTickMsg struct{ t time.Time }

type screen int

const (
	screenWelcome screen = iota
	screenList
	screenNew
	screenEdit
	screenDelete
	screenEarlyTrain
	screenTrain
	screenCongrats
	screenLevel
	screenSettings
)

type vaultStep int

const (
	vaultStepWarn vaultStep = iota
	vaultStepPassword
	vaultStepConfirm
	vaultStepMenu
	vaultStepChangePassword
	vaultStepChangeConfirm
	vaultStepDecryptWarn
)

type settingsCategory int

const (
	settingsCategoryVault settingsCategory = iota
	settingsCategoryLevels
)

type Model struct {
	db     *store.DB
	eng    *engine.Engine
	cfg    config.Config
	levels []levels.Level
	beeper beep.Beep

	width, height    int
	screen           screen
	errMsg           string
	welcomeDeadline  time.Time
	welcomeRemaining int

	// list
	trainers []store.Trainer
	cursor   int

	// new
	newInputs       [2]textinput.Model
	newFocus        int
	newLevelIndex   int
	newShowPassword bool

	// edit
	editTrainer    *store.Trainer
	editInputs     [1]textinput.Model
	editFocus      int
	editLevelIndex int

	// delete
	deleteIndex int

	// early train warning
	earlyTrainer *store.Trainer

	// train
	trainer    *store.Trainer
	counts     bool
	mask       engine.Mask
	congrats   string
	input      string
	hint       bool
	hintUntil  time.Time
	attempt    int
	delaying   bool
	delayUntil time.Time
	countdown  int
	tStart     time.Time
	tErrors    int

	// level offer
	levelTrainer *store.Trainer
	levelOffer   int

	// settings / vault
	settingsCat  settingsCategory
	vaultStep    vaultStep
	vaultPw      textinput.Model
	vaultConfirm textinput.Model
	vaultShowPw  bool
	vaultChoice  int
	vaultErrMsg  string

	// settings / levels
	settingsTabsFocused bool
	settingsFocus       int
	settingsExpanded    []bool
	settingsEditing     bool
	settingsEditInput   textinput.Model
	settingsEditIdx     int
	settingsEditField   string
	settingsEditErr     string
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	lvl, err := levels.Load()
	if err != nil {
		return err
	}
	for {
		db, err := store.New()
		if err != nil {
			return err
		}
		exists, err := db.VaultExists()
		if err != nil {
			return err
		}
		// If the user already dismissed the initial encryption prompt,
		// start without a vault and do not ask again.
		if !exists && cfg.PromptedForVault {
			m := New(db, cfg, lvl)
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err = p.Run()
			return err
		}
		uModel := newUnlockModel(db, !exists)
		pUnlock := tea.NewProgram(uModel, tea.WithAltScreen())
		final, err := pUnlock.Run()
		if err != nil {
			return err
		}
		um := final.(unlockModel)
		if um.reset {
			continue
		}
		if !um.skipped && um.result == nil {
			return errors.New("vault not unlocked")
		}
		if !um.skipped {
			db.SetVault(um.result)
		}
		cfg.PromptedForVault = true
		if err := config.Save(cfg); err != nil {
			return err
		}
		m := New(db, cfg, lvl)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err = p.Run()
		return err
	}
}

func New(db *store.DB, cfg config.Config, lvl []levels.Level) Model {
	m := Model{
		db:               db,
		eng:              engine.New(lvl),
		cfg:              cfg,
		levels:           lvl,
		beeper:           beep.Terminal{},
		screen:           screenWelcome,
		welcomeDeadline:  time.Now().Add(2 * time.Second),
		welcomeRemaining: 2,
	}
	m.newInputs[0] = textinput.New()
	m.newInputs[0].Placeholder = "Label"
	m.newInputs[0].Focus()
	m.newInputs[1] = textinput.New()
	m.newInputs[1].Placeholder = "Password"
	m.newInputs[1].EchoMode = textinput.EchoPassword
	m.newLevelIndex = 0

	m.vaultPw = textinput.New()
	m.vaultPw.Placeholder = "master password"
	m.vaultPw.EchoMode = textinput.EchoPassword
	m.vaultPw.EchoCharacter = '•'
	m.vaultPw.CharLimit = 64

	m.vaultConfirm = textinput.New()
	m.vaultConfirm.Placeholder = "confirm password"
	m.vaultConfirm.EchoMode = textinput.EchoPassword
	m.vaultConfirm.EchoCharacter = '•'
	m.vaultConfirm.CharLimit = 64

	m.settingsEditInput = textinput.New()
	m.settingsEditInput.CharLimit = 64
	m.settingsExpanded = make([]bool, len(lvl))

	m.loadTrainers()
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return welcomeTickMsg{t}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - 6
		m.height = msg.Height - 1
		if m.width < 0 {
			m.width = 0
		}
		if m.height < 0 {
			m.height = 0
		}
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.screen == screenWelcome {
			return m.updateWelcome(msg)
		}
		if m.screen == screenList {
			return m.updateList(msg)
		}
		if m.screen == screenNew {
			return m.updateNew(msg)
		}
		if m.screen == screenEdit {
			return m.updateEdit(msg)
		}
		if m.screen == screenDelete {
			return m.updateDelete(msg)
		}
		if m.screen == screenEarlyTrain {
			return m.updateEarlyTrain(msg)
		}
		if m.screen == screenTrain {
			return m.updateTrain(msg)
		}
		if m.screen == screenCongrats {
			return m.updateCongrats(msg)
		}
		if m.screen == screenLevel {
			return m.updateLevel(msg)
		}
		if m.screen == screenSettings {
			return m.updateSettings(msg)
		}
	case welcomeTickMsg:
		if m.screen != screenWelcome {
			return m, nil
		}
		remaining := time.Until(m.welcomeDeadline)
		if remaining <= 0 {
			m.screen = screenList
			m.loadTrainers()
			return m, nil
		}
		m.welcomeRemaining = int(remaining.Seconds()) + 1
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return welcomeTickMsg{t}
		})
	case tickMsg:
		if m.screen == screenTrain {
			return m.updateTrainTick(msg)
		}
	}
	return m, nil
}

func (m Model) View() string {
	var inner string
	switch m.screen {
	case screenWelcome:
		inner = m.welcomeView()
	case screenList:
		inner = m.listView()
	case screenNew:
		inner = m.newView()
	case screenEdit:
		inner = m.editView()
	case screenDelete:
		inner = m.deleteView()
	case screenEarlyTrain:
		inner = m.earlyView()
	case screenTrain:
		inner = m.trainView()
	case screenCongrats:
		inner = m.congratsView()
	case screenLevel:
		inner = m.levelView()
	case screenSettings:
		inner = m.settingsView()
	default:
		inner = "unknown screen"
	}
	return lipgloss.NewStyle().Padding(1, 3, 0, 3).Render(inner)
}

func (m *Model) loadTrainers() {
	list, err := m.db.ListTrainers()
	if err != nil {
		m.errMsg = err.Error()
		return
	}
	m.trainers = list
	if m.cursor >= len(m.trainers) {
		m.cursor = 0
	}
}

func (m *Model) isDue(t store.Trainer) bool {
	status, _ := m.eng.Availability(t)
	return status == "due"
}

func (m *Model) canCount(t store.Trainer) bool {
	_, canCount := m.eng.Availability(t)
	return canCount
}

func (m *Model) levelColor(n int) string {
	for _, l := range m.levels {
		if l.Number == n {
			return l.Color
		}
	}
	return "#fff"
}

func (m *Model) findLevelIndex(n int) int {
	for i, l := range m.levels {
		if l.Number == n {
			return i
		}
	}
	return 0
}

func renderTitle(text string, bg ...lipgloss.Color) string {
	c := glow.Fuchsia
	if len(bg) > 0 {
		c = bg[0]
	}
	return lipgloss.NewStyle().
		Bold(true).
		Background(c).
		Foreground(glow.Cream).
		Padding(0, 1).
		Render(text)
}

func rainbow(text string) string {
	colors := []string{
		"#FF0000", "#FF7F00", "#FFFF00",
		"#00FF00", "#0000FF", "#4B0082", "#9400D3",
	}
	var b strings.Builder
	i := 0
	for _, r := range text {
		if r == ' ' {
			b.WriteRune(r)
		} else {
			c := colors[i%len(colors)]
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(string(r)))
			i++
		}
	}
	return b.String()
}

func dimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(glow.Dim)
}

func (m *Model) resetNew() {
	m.newInputs[0].SetValue("")
	m.newInputs[1].SetValue("")
	m.newInputs[0].Focus()
	m.newInputs[1].Blur()
	m.newInputs[1].EchoMode = textinput.EchoPassword
	m.newShowPassword = false
	m.newFocus = 0
	m.newLevelIndex = 0
}

func (m Model) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if s == "q" {
		return m, tea.Quit
	}
	if s == "enter" || s == " " {
		m.screen = screenList
		m.loadTrainers()
	}
	return m, nil
}

func (m Model) welcomeView() string {
	due := 0
	for _, t := range m.trainers {
		if m.isDue(t) {
			due++
		}
	}

	w, h := m.width, m.height
	if w == 0 {
		w = consts.DefaultTermWidth
	}
	if h == 0 {
		h = consts.DefaultTermHeight
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(glow.Fuchsia).
		Render(fmt.Sprintf("Welcome to %s", consts.AppName))

	var warning string
	if !m.db.HasVault() {
		warning = dimStyle().Render("Not encrypted")
	}

	footer := fmt.Sprintf("Press %s to skip, %s to quit.\n%s",
		rainbow("Enter"), lipgloss.NewStyle().Foreground(glow.Red).Render("q"),
		dimStyle().Render(fmt.Sprintf("Skipping in %d...", m.welcomeRemaining)))

	body := lipgloss.NewStyle().
		Bold(true).
		Render(fmt.Sprintf("Pending sessions: %d\nTotal trainers: %d",
			due, len(m.trainers)))

	content := title
	if warning != "" {
		content += "\n" + warning
	}
	content += "\n\n" + body + "\n\n" + footer
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch s {
	case "q":
		return m, tea.Quit
	case "r":
		m.loadTrainers()
	case "n":
		m.resetNew()
		m.screen = screenNew
	case "e":
		if len(m.trainers) > 0 {
			m.startEditing(&m.trainers[m.cursor])
		}
	case "d":
		if len(m.trainers) > 0 {
			m.deleteIndex = m.cursor
			m.screen = screenDelete
		}
	case "enter":
		if len(m.trainers) > 0 {
			t := &m.trainers[m.cursor]
			if m.canCount(*t) {
				m.startTraining(t, true)
			} else {
				m.earlyTrainer = t
				m.screen = screenEarlyTrain
			}
		}
	case "j", "down":
		if m.cursor < len(m.trainers)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.trainers) - 1
	case "s":
		m.resetSettings()
		m.screen = screenSettings
	}
	return m, nil
}

func (m Model) listView() string {
	var list strings.Builder
	if len(m.trainers) == 0 {
		list.WriteString("No trainers. Press n to add one.")
	} else {
		for i, t := range m.trainers {
			label := t.Label
			if i == m.cursor {
				label = "> " + label
			} else {
				label = "  " + label
			}
			color := m.levelColor(t.Level)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
			if i == m.cursor {
				style = style.Bold(true)
			}
			line := style.Render(label)
			if m.isDue(t) {
				line += lipgloss.NewStyle().Foreground(glow.Red).Render(" [due]")
			}
			list.WriteString(line)
			list.WriteString("\n")
		}
	}
	if m.errMsg != "" {
		list.WriteString("\n")
		list.WriteString(lipgloss.NewStyle().Foreground(glow.Red).Render("error: " + m.errMsg))
	}

	title := renderTitle("passmem")
	if !m.db.HasVault() {
		title += "   " + dimStyle().Render("Not encrypted")
	}
	var right string
	if len(m.trainers) > 0 {
		right = strings.Repeat("\n", m.cursor) + m.trainerDetailsView(m.trainers[m.cursor])
	}
	body := title + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, list.String(), "  ", right)

	guide := "n: new  e: edit  d: delete  r: refresh  s: settings  enter: train  q: quit"
	footer := dimStyle().Render(guide)

	if m.height > 0 {
		bodyLines := strings.Count(body, "\n") + 1
		footerLines := strings.Count(footer, "\n") + 1
		gap := m.height - bodyLines - footerLines
		if gap < 0 {
			gap = 0
		}
		body += strings.Repeat("\n", gap) + footer
	} else {
		body += "\n\n" + footer
	}

	return body
}

func (m Model) updateNew(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	s := msg.String()

	if s == "esc" {
		m.screen = screenList
		return m, nil
	}
	if s == "ctrl+s" {
		m.newShowPassword = !m.newShowPassword
		if m.newShowPassword {
			m.newInputs[1].EchoMode = textinput.EchoNormal
		} else {
			m.newInputs[1].EchoMode = textinput.EchoPassword
		}
		return m, nil
	}
	if s == "tab" || s == "shift+tab" {
		dir := 1
		if s == "shift+tab" {
			dir = -1
		}
		if m.newFocus < 2 {
			m.newInputs[m.newFocus].Blur()
		}
		m.newFocus = (m.newFocus + dir + 3) % 3
		if m.newFocus < 2 {
			m.newInputs[m.newFocus].Focus()
		}
		return m, nil
	}
	if m.newFocus == 2 {
		switch s {
		case "k", "up":
			if m.newLevelIndex > 0 {
				m.newLevelIndex--
			}
		case "j", "down":
			if m.newLevelIndex < len(m.levels)-1 {
				m.newLevelIndex++
			}
		}
		return m, nil
	}

	if s == "enter" {
		if err := m.createTrainer(); err != nil {
			m.errMsg = err.Error()
		} else {
			m.errMsg = ""
			m.screen = screenList
			m.loadTrainers()
			m.resetNew()
		}
		return m, nil
	}

	i := m.newFocus
	if i >= 0 && i < 2 {
		m.newInputs[i], _ = m.newInputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) createTrainer() error {
	label := strings.TrimSpace(m.newInputs[0].Value())
	password := m.newInputs[1].Value()
	if label == "" {
		return errors.New("label is required")
	}
	if password == "" {
		return errors.New("password is required")
	}
	level := m.levels[m.newLevelIndex].Number
	_, err := m.db.CreateTrainer(label, password, level)
	return err
}

func (m Model) blurPreview(password string, levelIndex int) string {
	if password == "" {
		return ""
	}
	level := m.levels[levelIndex]
	mask, err := m.eng.Preview(level, password)
	if err != nil {
		return ""
	}
	preview := lipgloss.NewStyle().Foreground(glow.DullFuchsia).Render(mask.Blurred)
	return "\n" + dimStyle().Render("Blur preview: ") + preview
}

func (m Model) newView() string {
	var b strings.Builder
	b.WriteString(renderTitle("New trainer"))
	b.WriteString("\n\n")

	for i := range m.newInputs {
		name := []string{"Label", "Password"}[i]
		prefix := "  "
		if i == m.newFocus {
			prefix = "> "
		}
		b.WriteString(prefix + name + ":\n")
		b.WriteString(m.newInputs[i].View())
		b.WriteString("\n")
	}

	check := "[ ]"
	if m.newShowPassword {
		check = "[x]"
	}
	b.WriteString(dimStyle().Render("  " + check + " Show password (ctrl+s to toggle)"))
	b.WriteString("\n\n")

	if m.newFocus == 2 {
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(glow.Fuchsia).
			Render("> Familiarity level (j/k to select)"))
	} else {
		b.WriteString(dimStyle().Render("  Familiarity level"))
	}
	b.WriteString("\n")
	descWidth := m.width - 26
	if descWidth < 20 {
		descWidth = 20
	}
	for i, l := range m.levels {
		var left string
		if i == m.newLevelIndex {
			left = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color(l.Color)).
				Foreground(lipgloss.Color("#fff")).
				Width(18).
				Render(fmt.Sprintf("> Level %d", l.Number))
		} else {
			left = lipgloss.NewStyle().
				Foreground(lipgloss.Color(l.Color)).
				Width(18).
				Render(fmt.Sprintf("  Level %d", l.Number))
		}
		if i == m.newLevelIndex {
			right := lipgloss.NewStyle().
				Foreground(glow.Dim).
				Width(descWidth).
				Render(l.Description)
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))
		} else {
			b.WriteString(left)
		}
		b.WriteString("\n")
	}
	b.WriteString(m.blurPreview(m.newInputs[1].Value(), m.newLevelIndex))

	if m.errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(glow.Red).Render("error: " + m.errMsg))
	}

	body := strings.TrimSuffix(b.String(), "\n")
	footer := dimStyle().Render("tab: switch focus • j/k: choose level • enter: save • esc: cancel")

	if m.height > 0 {
		bodyLines := strings.Count(body, "\n") + 1
		footerLines := strings.Count(footer, "\n") + 1
		gap := m.height - bodyLines - footerLines
		if gap < 0 {
			gap = 0
		}
		body += strings.Repeat("\n", gap) + footer
	} else {
		body += "\n\n" + footer
	}
	return body
}

func (m *Model) startEditing(t *store.Trainer) {
	m.editTrainer = t
	m.editInputs[0] = textinput.New()
	m.editInputs[0].Placeholder = "Label"
	m.editInputs[0].SetValue(t.Label)
	m.editInputs[0].Focus()
	m.editLevelIndex = m.findLevelIndex(t.Level)
	m.editFocus = 0
	m.errMsg = ""
	m.screen = screenEdit
}

func (m Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()

	if s == "esc" {
		m.screen = screenList
		m.editTrainer = nil
		return m, nil
	}
	if s == "tab" || s == "shift+tab" {
		dir := 1
		if s == "shift+tab" {
			dir = -1
		}
		if m.editFocus < 1 {
			m.editInputs[m.editFocus].Blur()
		}
		m.editFocus = (m.editFocus + dir + 2) % 2
		if m.editFocus < 1 {
			m.editInputs[m.editFocus].Focus()
		}
		return m, nil
	}
	if m.editFocus == 1 {
		switch s {
		case "k", "up":
			if m.editLevelIndex > 0 {
				m.editLevelIndex--
			}
		case "j", "down":
			if m.editLevelIndex < len(m.levels)-1 {
				m.editLevelIndex++
			}
		}
		return m, nil
	}
	if s == "enter" {
		if err := m.saveEdit(); err != nil {
			m.errMsg = err.Error()
		} else {
			m.errMsg = ""
			m.loadTrainers()
			m.screen = screenList
			m.editTrainer = nil
		}
		return m, nil
	}

	if m.editFocus == 0 {
		m.editInputs[0], _ = m.editInputs[0].Update(msg)
	}
	return m, nil
}

func (m *Model) saveEdit() error {
	if m.editTrainer == nil {
		return errors.New("no trainer selected")
	}
	label := strings.TrimSpace(m.editInputs[0].Value())
	if label == "" {
		return errors.New("label is required")
	}
	m.editTrainer.Label = label
	newLevel := m.levels[m.editLevelIndex].Number
	if m.editTrainer.Level != newLevel {
		m.editTrainer.Level = newLevel
		m.editTrainer.SessionsAtLevel = 0
	}
	return m.db.UpdateTrainer(*m.editTrainer)
}

func (m Model) editView() string {
	var b strings.Builder
	b.WriteString(renderTitle("Edit trainer"))
	b.WriteString("\n\n")

	prefix := "> "
	if m.editFocus != 0 {
		prefix = "  "
	}
	b.WriteString(prefix + "Label:\n")
	b.WriteString(m.editInputs[0].View())
	b.WriteString("\n\n")

	if m.editFocus == 1 {
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(glow.Fuchsia).
			Render("> Familiarity level (j/k to select)"))
	} else {
		b.WriteString(dimStyle().Render("  Familiarity level"))
	}
	b.WriteString("\n")

	descWidth := m.width - 26
	if descWidth < 20 {
		descWidth = 20
	}
	for i, l := range m.levels {
		var left string
		if i == m.editLevelIndex {
			left = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color(l.Color)).
				Foreground(lipgloss.Color("#fff")).
				Width(18).
				Render(fmt.Sprintf("> Level %d", l.Number))
		} else {
			left = lipgloss.NewStyle().
				Foreground(lipgloss.Color(l.Color)).
				Width(18).
				Render(fmt.Sprintf("  Level %d", l.Number))
		}
		if i == m.editLevelIndex {
			right := lipgloss.NewStyle().
				Foreground(glow.Dim).
				Width(descWidth).
				Render(l.Description)
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))
		} else {
			b.WriteString(left)
		}
		b.WriteString("\n")
	}
	if m.editTrainer != nil {
		b.WriteString(m.blurPreview(m.editTrainer.Password, m.editLevelIndex))
	}

	if m.errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(glow.Red).Render("error: " + m.errMsg))
	}

	body := strings.TrimSuffix(b.String(), "\n")
	footer := dimStyle().Render("tab: switch focus • j/k: choose level • enter: save • esc: cancel")

	if m.height > 0 {
		bodyLines := strings.Count(body, "\n") + 1
		footerLines := strings.Count(footer, "\n") + 1
		gap := m.height - bodyLines - footerLines
		if gap < 0 {
			gap = 0
		}
		body += strings.Repeat("\n", gap) + footer
	} else {
		body += "\n\n" + footer
	}
	return body
}

func (m Model) updateEarlyTrain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if s == "y" {
		m.startTraining(m.earlyTrainer, false)
		m.earlyTrainer = nil
	} else if s == "n" || s == "esc" {
		m.earlyTrainer = nil
		m.screen = screenList
	}
	return m, nil
}

func (m Model) earlyView() string {
	label := m.earlyTrainer.Label

	availableAt, _, _, _ := m.eng.Schedule(*m.earlyTrainer)
	var availableIn string
	d := time.Until(availableAt)
	if d > 0 {
		availableIn = dimStyle().Render(fmt.Sprintf("Available in %s", formatDuration(d)))
	}

	raw := "A counted session was completed recently. You need to wait for the interval to pass before the next session counts. You can still practice, but it will not affect progress."
	wrap := 40
	if m.width > 0 && m.width < wrap+10 {
		wrap = m.width - 10
	}
	if wrap < 30 {
		wrap = 30
	}
	explanation := wrapWords(raw, wrap)

	resting := lipgloss.NewStyle().Foreground(glow.LightBlue).Render("resting")
	yesLine := lipgloss.NewStyle().Foreground(glow.Green).Render("[y]es — practice anyway")
	noLine := lipgloss.NewStyle().Foreground(glow.Red).Render("[n]o — go back")

	body := fmt.Sprintf("%s is %s right now which means that %s\n\n%s\n\n%s\n%s",
		label, resting, explanation,
		availableIn,
		yesLine,
		noLine)
	return renderTitle("Not available yet") + "\n\n" + body
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func wrapWords(text string, width int) string {
	words := strings.Fields(text)
	var b strings.Builder
	var line string
	for _, w := range words {
		if line != "" && len(line)+1+len(w) > width {
			b.WriteString(line)
			b.WriteString("\n")
			line = w
		} else {
			if line != "" {
				line += " "
			}
			line += w
		}
	}
	if line != "" {
		b.WriteString(line)
	}
	return b.String()
}

func (m Model) updateDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if s == "y" {
		id := m.trainers[m.deleteIndex].ID
		if err := m.db.DeleteTrainer(id); err != nil {
			m.errMsg = err.Error()
		}
		m.loadTrainers()
		m.screen = screenList
	} else if s == "n" || s == "esc" {
		m.screen = screenList
	}
	return m, nil
}

func (m Model) deleteView() string {
	label := m.trainers[m.deleteIndex].Label
	return renderTitle("Delete trainer") + "\n\n" + fmt.Sprintf("Delete %q?\n\n[y]es / [n]o", label)
}

func (m Model) updateCongrats(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if s != "" {
		if m.levelTrainer != nil {
			m.screen = screenLevel
		} else {
			m.loadTrainers()
			m.screen = screenList
		}
		m.congrats = ""
	}
	return m, nil
}

func (m Model) congratsView() string {
	return renderTitle("Session complete") + "\n\n" + m.congrats + "\n\nPress any key to continue."
}

func (m Model) trainerDetailsView(t store.Trainer) string {
	level, ok := m.eng.LevelConfig(t.Level)
	if !ok {
		return "unknown level"
	}

	needed := level.RequiredSessionsToProgress - t.SessionsAtLevel
	if needed < 0 {
		needed = 0
	}

	availableAt, dueAt, status, _ := m.eng.Schedule(t)
	now := time.Now()

	var availableLine, dueLine string
	if availableAt.After(now) {
		availableLine = fmt.Sprintf("Available in %s", formatDuration(availableAt.Sub(now)))
	} else {
		availableLine = "Available now"
	}
	if dueAt.After(now) {
		dueLine = fmt.Sprintf("Due in %s", formatDuration(dueAt.Sub(now)))
	} else {
		dueLine = fmt.Sprintf("Overdue by %s", formatDuration(now.Sub(dueAt)))
	}

	statusText := status
	switch status {
	case "resting":
		statusText = lipgloss.NewStyle().Foreground(glow.LightBlue).Render(status)
	case "available":
		statusText = lipgloss.NewStyle().Foreground(glow.Green).Render(status)
	case "due":
		statusText = lipgloss.NewStyle().Foreground(glow.Red).Render(status)
	}

	sessions := fmt.Sprintf("Sessions at level: %d / %d", t.SessionsAtLevel, level.RequiredSessionsToProgress) +
		dimStyle().Render(fmt.Sprintf("  (%d to progress)", needed))

	lines := []string{
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(level.Color)).
			Bold(true).
			Render(fmt.Sprintf("Familiarity level %d", t.Level)),
		sessions,
		availableLine,
	}
	if status != "resting" {
		lines = append(lines, dueLine)
	}
	lines = append(lines,
		fmt.Sprintf("Status: %s", statusText),
		dimStyle().Render(fmt.Sprintf("Total sessions: %d", t.TotalSessions)),
		dimStyle().Render(fmt.Sprintf("Created: %s", t.CreatedAt.Format("2006-01-02"))),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(glow.Fuchsia).
		Padding(1, 2).
		Render(renderTitle(t.Label, glow.Red) + "\n\n" + strings.Join(lines, "\n"))
}

func (m *Model) startTraining(t *store.Trainer, counts bool) {
	m.trainer = t
	m.counts = counts
	m.input = ""
	m.hint = false
	m.attempt = 0
	m.delaying = false
	m.tStart = time.Now()
	m.tErrors = 0
	m.errMsg = ""
	mask, err := m.eng.MaskFor(t)
	if err != nil {
		m.errMsg = err.Error()
		m.screen = screenList
		return
	}
	m.mask = mask
	m.screen = screenTrain
}
