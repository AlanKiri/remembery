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
	"github.com/alankiri/password-memorizer-tui/internal/engine"
	"github.com/alankiri/password-memorizer-tui/internal/levels"
	"github.com/alankiri/password-memorizer-tui/internal/store"
)

type screen int

const (
	screenWelcome screen = iota
	screenList
	screenNew
	screenDelete
	screenTrain
	screenLevel
)

type Model struct {
	db     *store.DB
	eng    *engine.Engine
	cfg    config.Config
	levels []levels.Level
	beeper beep.Beep

	width, height int
	screen        screen
	errMsg        string

	// list
	trainers []store.Trainer
	cursor   int

	// new
	newInputs     [2]textinput.Model
	newFocus      int
	newLevelIndex int

	// delete
	deleteIndex int

	// train
	trainer    *store.Trainer
	mask       engine.Mask
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
		db:     db,
		eng:    engine.New(lvl),
		cfg:    cfg,
		levels: lvl,
		beeper: beep.Terminal{},
		screen: screenWelcome,
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
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
		if m.screen == screenDelete {
			return m.updateDelete(msg)
		}
		if m.screen == screenTrain {
			return m.updateTrain(msg)
		}
		if m.screen == screenLevel {
			return m.updateLevel(msg)
		}
	case tickMsg:
		if m.screen == screenTrain {
			return m.updateTrainTick(msg)
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.screen {
	case screenWelcome:
		return m.welcomeView()
	case screenList:
		return m.listView()
	case screenNew:
		return m.newView()
	case screenDelete:
		return m.deleteView()
	case screenTrain:
		return m.trainView()
	case screenLevel:
		return m.levelView()
	}
	return "unknown screen"
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

func (m *Model) resetNew() {
	m.newInputs[0].SetValue("")
	m.newInputs[1].SetValue("")
	m.newInputs[0].Focus()
	m.newInputs[1].Blur()
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
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Render("Welcome to passmem")
	body := fmt.Sprintf("%s\n\nPending sessions: %d\nTotal trainers: %d\nStudy days: %v\n\nPress Enter to start, q to quit.",
		title, due, len(m.trainers), m.cfg.Welcome.StudyDays)
	return body
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
	case "d":
		if len(m.trainers) > 0 {
			m.deleteIndex = m.cursor
			m.screen = screenDelete
		}
	case "enter":
		if len(m.trainers) > 0 {
			m.startTraining(&m.trainers[m.cursor])
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
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("passmem — [n]ew [d]elete [r]efresh [q]uit"))
	b.WriteString("\n\n")
	if len(m.trainers) == 0 {
		b.WriteString("No trainers. Press n to add one.")
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
				line += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(" [due]")
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("error: " + m.errMsg))
	}
	return b.String()
}

func (m Model) updateNew(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	s := msg.String()

	if s == "esc" {
		m.screen = screenList
		return m, nil
	}
	if s == "tab" {
		if m.newFocus < 2 {
			m.newInputs[m.newFocus].Blur()
		}
		m.newFocus = (m.newFocus + 1) % 3
		if m.newFocus < 2 {
			m.newInputs[m.newFocus].Focus()
		}
		return m, nil
	}
	if m.newFocus == 2 {
		switch s {
		case "up", "k":
			if m.newLevelIndex > 0 {
				m.newLevelIndex--
			}
		case "down", "j":
			if m.newLevelIndex < len(m.levels)-1 {
				m.newLevelIndex++
			}
		case "enter":
			m.newFocus = 0
			m.newInputs[0].Focus()
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
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("New trainer"))
	b.WriteString("\n\n")

	for i := range m.newInputs {
		name := []string{"Label", "Password"}[i]
		prefix := "  "
		if i == m.newFocus {
			prefix = "> "
		}
		b.WriteString(prefix + name + ":\n")
		b.WriteString(m.newInputs[i].View())
		b.WriteString("\n\n")
	}

	b.WriteString("Starting level:\n")
	for i, l := range m.levels {
		prefix := "  "
		if m.newFocus == 2 && i == m.newLevelIndex {
			prefix = "> "
		} else if i == m.newLevelIndex {
			prefix = "* "
		}
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(l.Color))
		b.WriteString(prefix + style.Render(fmt.Sprintf("Level %d: %s", l.Number, l.Description)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render("tab: switch focus • up/down or j/k: choose level • enter: save • esc: cancel"))
	if m.errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("error: " + m.errMsg))
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
	return fmt.Sprintf("Delete trainer %q?\n\n[y]es / [n]o", label)
}

func (m *Model) startTraining(t *store.Trainer) {
	m.trainer = t
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
