package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/remembery/internal/engine"
	"github.com/alankiri/remembery/internal/levels"
	"github.com/alankiri/remembery/internal/store"
	"github.com/alankiri/remembery/internal/ui/common"
	"github.com/alankiri/remembery/internal/ui/screen"
	"github.com/alankiri/remembery/internal/ui/styles"
)

// ListModel manages the trainer list and selection.
type ListModel struct {
	db       *store.DB
	eng      *engine.Engine
	levels   []levels.Level
	Trainers []store.Trainer
	cursor   int
}

// NewListModel creates a new list model.
func NewListModel(db *store.DB, eng *engine.Engine, levels []levels.Level) ListModel {
	return ListModel{
		db:     db,
		eng:    eng,
		levels: levels,
	}
}

// Init is a no-op for the list screen.
func (m ListModel) Init() tea.Cmd {
	return nil
}

// LoadTrainers refreshes the trainer list from the database.
func (m *ListModel) LoadTrainers() error {
	list, err := m.db.ListTrainers()
	if err != nil {
		return err
	}
	m.Trainers = list
	if m.cursor >= len(m.Trainers) {
		m.cursor = 0
	}
	return nil
}

func (m ListModel) IsDue(t store.Trainer) bool {
	status, _ := m.eng.Availability(t)
	return status == "due"
}

func (m ListModel) canCount(t store.Trainer) bool {
	_, canCount := m.eng.Availability(t)
	return canCount
}

// Update handles navigation and actions in the trainer list.
func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	s := key.String()
	switch s {
	case "q":
		return m, tea.Quit
	case "r":
		if err := m.LoadTrainers(); err != nil {
			return m, common.SetErr(err.Error())
		}
		return m, nil
	case "n":
		return m, screen.ChangeScreen(screen.ScreenNew)
	case "e":
		if len(m.Trainers) > 0 {
			return m, common.StartEdit(&m.Trainers[m.cursor])
		}
	case "d":
		if len(m.Trainers) > 0 {
			return m, common.StartDelete(&m.Trainers[m.cursor])
		}
	case "enter":
		if len(m.Trainers) > 0 {
			t := &m.Trainers[m.cursor]
			if m.canCount(*t) {
				return m, common.StartTrain(t, true)
			}
			return m, common.ShowEarly(t)
		}
	case "j", "down":
		if m.cursor < len(m.Trainers)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.Trainers) - 1
	case "v":
		return m, screen.ChangeScreen(screen.ScreenVault)
	}
	return m, nil
}

// View renders the trainer list with details for the selected item.
func (m ListModel) View(w, h int, errMsg string) string {
	var list strings.Builder
	if len(m.Trainers) == 0 {
		list.WriteString("No Trainers. ")
		list.WriteString(styles.DimStyle.Render("\nPress n to add one."))
	} else {
		for i, t := range m.Trainers {
			label := t.Label
			if i == m.cursor {
				label = "> " + label
			} else {
				label = "  " + label
			}
			color := common.LevelColor(m.levels, t.Level)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
			if i == m.cursor {
				style = style.Bold(true)
			}
			line := style.Render(label)
			if m.IsDue(t) {
				line += lipgloss.NewStyle().Foreground(styles.Glow.Red).Render(" [due]")
			}
			list.WriteString(line)
			list.WriteString("\n")
		}
	}
	if errMsg != "" {
		list.WriteString("\n")
		list.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Red).Render("error: " + errMsg))
	}

	title := styles.RenderTitle("List")
	if !m.db.HasVault() {
		title += "   " + styles.DimStyle.Render("Not encrypted")
	}
	var right string
	if len(m.Trainers) > 0 {
		right = strings.Repeat("\n", m.cursor) + m.trainerDetailsView(m.Trainers[m.cursor])
	}
	body := title + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, list.String(), "  ", right)

	guide := "n: new  e: edit  d: delete  r: refresh  v: vault  enter: train  q: quit"
	footer := styles.DimStyle.Render(guide)

	if h > 0 {
		bodyLines := strings.Count(body, "\n") + 1
		footerLines := strings.Count(footer, "\n") + 1
		gap := h - bodyLines - footerLines
		if gap < 0 {
			gap = 0
		}
		body += strings.Repeat("\n", gap) + footer
	} else {
		body += "\n\n" + footer
	}

	return body
}

func (m ListModel) trainerDetailsView(t store.Trainer) string {
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
		availableLine = fmt.Sprintf("Available in %s", common.FormatDuration(availableAt.Sub(now)))
	} else {
		availableLine = "Available now"
	}
	if dueAt.After(now) {
		dueLine = fmt.Sprintf("Due in %s", common.FormatDuration(dueAt.Sub(now)))
	} else {
		dueLine = fmt.Sprintf("Overdue by %s", common.FormatDuration(now.Sub(dueAt)))
	}

	statusText := status
	switch status {
	case "resting":
		statusText = lipgloss.NewStyle().Foreground(styles.Glow.LightBlue).Render(status)
	case "available":
		statusText = lipgloss.NewStyle().Foreground(styles.Glow.Green).Render(status)
	case "due":
		statusText = lipgloss.NewStyle().Foreground(styles.Glow.Red).Render(status)
	}

	sessions := fmt.Sprintf("Sessions at  level: %d / %d", t.SessionsAtLevel, level.RequiredSessionsToProgress) +
		styles.DimStyle.Render(fmt.Sprintf("  (%d to progress)", needed))

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
		styles.DimStyle.Render(fmt.Sprintf("Total  sessions: %d", t.TotalSessions)),
		styles.DimStyle.Render(fmt.Sprintf("Created: %s", t.CreatedAt.Format("2006-01-02"))),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Glow.Fuchsia).
		Padding(1, 2).
		Render(styles.RenderTitle(t.Label, styles.Glow.Red) + "\n\n" + strings.Join(lines, "\n"))
}
