package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/remembery/internal/store"
	"github.com/alankiri/remembery/internal/ui/screen"
	"github.com/alankiri/remembery/internal/ui/styles"
)

// DeleteModel handles confirming and executing a trainer deletion.
type DeleteModel struct {
	db      *store.DB
	trainer store.Trainer
	choice  int
}

// NewDeleteModel creates a delete model for the given trainer.
func NewDeleteModel(db *store.DB, t store.Trainer) DeleteModel {
	return DeleteModel{db: db, trainer: t, choice: 0}
}

// Init is a no-op for the delete confirmation screen.
func (m DeleteModel) Init() tea.Cmd {
	return nil
}

// Update handles list navigation and deletes the trainer when confirmed.
func (m DeleteModel) Update(msg tea.Msg) (DeleteModel, tea.Cmd) {
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
		if m.choice == 1 {
			if err := m.db.DeleteTrainer(m.trainer.ID); err != nil {
				return m, screen.ChangeScreen(screen.ScreenList, err.Error())
			}
		}
		return m, screen.ChangeScreen(screen.ScreenList)
	case "esc", "q":
		return m, screen.ChangeScreen(screen.ScreenList)
	}
	return m, nil
}

// View renders the delete confirmation list.
func (m DeleteModel) View(w, h int) string {
	var body strings.Builder
	body.WriteString(styles.RenderTitle("Delete trainer"))
	body.WriteString("\n\n")
	body.WriteString(fmt.Sprintf("Delete %q?\n\n", m.trainer.Label))

	options := []string{"No", "Yes"}
	for i, opt := range options {
		style := lipgloss.NewStyle().Foreground(styles.Glow.Cream)
		if i == m.choice {
			style = lipgloss.NewStyle().Bold(true).Foreground(styles.Glow.Blue)
		}
		body.WriteString(style.Render("  " + opt))
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
