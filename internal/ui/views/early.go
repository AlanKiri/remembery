package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/password-memorizer-tui/internal/engine"
	"github.com/alankiri/password-memorizer-tui/internal/store"
	"github.com/alankiri/password-memorizer-tui/internal/ui/common"
	"github.com/alankiri/password-memorizer-tui/internal/ui/screen"
	"github.com/alankiri/password-memorizer-tui/internal/ui/styles"
)

// EarlyModel shows the early-training warning and lets the user practice anyway.
type EarlyModel struct {
	eng     *engine.Engine
	trainer *store.Trainer
}

// NewEarlyModel creates an early-train warning model for the given trainer.
func NewEarlyModel(eng *engine.Engine, t *store.Trainer) EarlyModel {
	return EarlyModel{eng: eng, trainer: t}
}

// Init is a no-op for the early training warning screen.
func (m EarlyModel) Init() tea.Cmd {
	return nil
}

// Update handles choosing to practice anyway or going back.
func (m EarlyModel) Update(msg tea.Msg) (EarlyModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	s := key.String()
	if s == "y" {
		return m, common.StartTrain(m.trainer, false)
	}
	if s == "n" || s == "esc" {
		return m, screen.ChangeScreen(screen.ScreenList)
	}
	return m, nil
}

// View renders the early training warning.
func (m EarlyModel) View(w, h int) string {
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
	yesLine := lipgloss.NewStyle().Foreground(styles.Glow.Green).Render("[y]es — practice anyway")
	noLine := lipgloss.NewStyle().Foreground(styles.Glow.Red).Render("[n]o — go back")

	body := fmt.Sprintf("%s is %s right now which means that %s\n\n%s\n\n%s\n%s",
		label, resting, explanation,
		availableIn,
		yesLine,
		noLine)
	return styles.RenderTitle("Not available yet") + "\n\n" + body
}
