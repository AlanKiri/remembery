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
	Green       lipgloss.Color
	Red         lipgloss.Color
	Yellow      lipgloss.Color
	Cream       lipgloss.Color
	Dim         lipgloss.Color
	White       lipgloss.Color
}{
	Fuchsia:     lipgloss.Color("#EE6FF8"),
	DullFuchsia: lipgloss.Color("#F793FF"),
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
	db, err := store.New()
	if err != nil {
		return err
	}
	m := New(db, cfg, lvl)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
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
	if t.NextDue == nil {
		return true
	}
	return !t.NextDue.After(time.Now())
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

func renderTitle(text string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Background(glow.Fuchsia).
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

	footer := fmt.Sprintf("Press %s to skip, %s to quit.\n%s",
		rainbow("Enter"), lipgloss.NewStyle().Foreground(glow.Red).Render("q"),
		dimStyle().Render(fmt.Sprintf("Skipping in %d...", m.welcomeRemaining)))

	body := lipgloss.NewStyle().
		Bold(true).
		Render(fmt.Sprintf("Pending sessions: %d\nTotal trainers: %d",
			due, len(m.trainers)))

	content := title + "\n\n" + body + "\n\n" + footer
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
			if m.isDue(*t) {
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
	body := title + "\n\n" + list.String()

	footer := dimStyle().Render("n: new  e: edit  d: delete  r: refresh  enter: train  q: quit")

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
	red := lipgloss.NewStyle().Foreground(glow.Red)

	var dueIn string
	if m.earlyTrainer.NextDue != nil {
		d := time.Until(*m.earlyTrainer.NextDue)
		if d > 0 {
			dueIn = fmt.Sprintf("Due again in %s", formatDuration(d))
		}
	}

	body := fmt.Sprintf("%q is not %s yet.\n\n%s\n\nPracticing now will not count toward progress.\n\n[y]es, practice anyway / [n]o, go back",
		label, red.Render("due"), dueIn)
	return renderTitle("Not due yet") + "\n\n" + body
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
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
