package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/password-memorizer-tui/internal/consts"
	"github.com/alankiri/password-memorizer-tui/internal/ui/common"
	"github.com/alankiri/password-memorizer-tui/internal/ui/screen"
	"github.com/alankiri/password-memorizer-tui/internal/ui/styles"
)

// WelcomeModel is the child model for the welcome screen.
type WelcomeModel struct {
	due       int
	total     int
	hasVault  bool
	remaining int
	deadline  time.Time
}

// NewWelcomeModel creates a new welcome model with the given summary data.
func NewWelcomeModel(due, total int, hasVault bool) WelcomeModel {
	return WelcomeModel{
		due:       due,
		total:     total,
		hasVault:  hasVault,
		remaining: 2,
		deadline:  time.Now().Add(2 * time.Second),
	}
}

// Init starts the welcome countdown tick.
func (m WelcomeModel) Init() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return common.WelcomeTickMsg{T: t}
	})
}

// Update handles key and tick messages for the welcome screen.
func (m WelcomeModel) Update(msg tea.Msg) (WelcomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		if s == "q" {
			return m, tea.Quit
		}
		if s == "enter" || s == " " {
			return m, screen.ChangeScreen(screen.ScreenList)
		}
	case common.WelcomeTickMsg:
		remaining := time.Until(m.deadline)
		if remaining <= 0 {
			return m, screen.ChangeScreen(screen.ScreenList)
		}
		m.remaining = int(remaining.Seconds()) + 1
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return common.WelcomeTickMsg{T: t}
		})
	}
	return m, nil
}

// View renders the welcome screen centered in the terminal.
func (m WelcomeModel) View(w, h int) string {
	if w == 0 {
		w = consts.DefaultTermWidth
	}
	if h == 0 {
		h = consts.DefaultTermHeight
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Glow.Fuchsia).
		Render(fmt.Sprintf("Welcome to %s", consts.AppName))

	var warning string
	if !m.hasVault {
		warning = styles.DimStyle.Render("Not encrypted")
	}

	footer := fmt.Sprintf("Press %s to skip, %s to quit.\n%s",
		styles.Rainbow("Enter"),
		lipgloss.NewStyle().Foreground(styles.Glow.Red).Render("q"),
		styles.DimStyle.Render(fmt.Sprintf("Skipping in %d...", m.remaining)))

	body := lipgloss.NewStyle().
		Bold(true).
		Render(fmt.Sprintf("Pending  sessions: %d\nTotal  trainers: %d",
			m.due, m.total))

	content := title
	if warning != "" {
		content += "\n" + warning
	}
	content += "\n\n" + body + "\n\n" + footer
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}
