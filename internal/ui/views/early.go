package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/remembery/internal/engine"
	"github.com/alankiri/remembery/internal/store"
	"github.com/alankiri/remembery/internal/ui/common"
	"github.com/alankiri/remembery/internal/ui/screen"
	"github.com/alankiri/remembery/internal/ui/styles"
)

// EarlyModel shows the early-training warning and lets the user practice anyway.
type EarlyModel struct {
	eng     *engine.Engine
	trainer *store.Trainer
	choice  int
}

// NewEarlyModel creates an early-train warning model for the given trainer.
func NewEarlyModel(eng *engine.Engine, t *store.Trainer) EarlyModel {
	return EarlyModel{eng: eng, trainer: t, choice: 0}
}

// Init is a no-op for the early training warning screen.
func (m EarlyModel) Init() tea.Cmd {
	return nil
}

// Update handles list navigation and starts practice or goes back.
func (m EarlyModel) Update(msg tea.Msg) (EarlyModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	s := key.String()
	switch s {
	case "j", "down":
		if m.choice < 1 {
			m.choice++
		}
		return m, nil
	case "k", "up":
		if m.choice > 0 {
			m.choice--
		}
		return m, nil
	case "enter":
		if m.choice == 0 {
			return m, common.StartTrain(m.trainer, false)
		}
		return m, screen.ChangeScreen(screen.ScreenList)
	case "esc", "q":
		return m, screen.ChangeScreen(screen.ScreenList)
	}
	return m, nil
}

// View renders the early training warning with a yes/no list.
func (m EarlyModel) View(w, h int) string {
	var body strings.Builder
	body.WriteString(styles.RenderTitle("Not available yet"))
	body.WriteString("\n\n")

	label := m.trainer.Label
	availableAt, _, _, _ := m.eng.Schedule(*m.trainer)
	var availableIn string
	d := time.Until(availableAt)
	if d > 0 {
		availableIn = styles.DimStyle.Render(fmt.Sprintf("Available in %s", common.FormatDuration(d)))
	}

	explanation := fmt.Sprintf("%s\n%s\n%s",
		"A counted session was completed recently.",
		"You need to wait for the interval to pass before the next session counts.",
		"You can still practice, but it will not affect progress.")

	resting := lipgloss.NewStyle().Foreground(styles.Glow.LightBlue).Render("resting")
	body.WriteString(fmt.Sprintf("%s is %s right now.\n\n%s\n\n%s\n",
		label, resting, explanation, availableIn))

	options := []string{"Yes — practice anyway", "No — go back"}
	for i, opt := range options {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(styles.Glow.Cream)
		if i == m.choice {
			prefix = "> "
			style = lipgloss.NewStyle().Bold(true).Foreground(styles.Glow.Blue)
		}
		body.WriteString(style.Render(prefix + opt))
		body.WriteString("\n")
	}

	bodyStr := strings.TrimRight(body.String(), "\n")
	footer := "\n" + styles.DimStyle.Render("j/k: choose  enter: confirm  esc: cancel")

	if h > 0 {
		bodyLines := strings.Count(bodyStr, "\n") + 1
		footerLines := strings.Count(footer, "\n") + 1
		gap := h - bodyLines - footerLines
		if gap < 0 {
			gap = 0
		}
		return bodyStr + strings.Repeat("\n", gap) + footer
	}
	return bodyStr + "\n" + footer
}
