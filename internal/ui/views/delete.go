package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alankiri/password-memorizer-tui/internal/store"
	"github.com/alankiri/password-memorizer-tui/internal/ui/screen"
	"github.com/alankiri/password-memorizer-tui/internal/ui/styles"
)

// DeleteModel handles confirming and executing a trainer deletion.
type DeleteModel struct {
	db      *store.DB
	trainer store.Trainer
}

// NewDeleteModel creates a delete model for the given trainer.
func NewDeleteModel(db *store.DB, t store.Trainer) DeleteModel {
	return DeleteModel{db: db, trainer: t}
}

// Init is a no-op for the delete confirmation screen.
func (m DeleteModel) Init() tea.Cmd {
	return nil
}

// Update handles yes/no confirmation and deletes the trainer on yes.
func (m DeleteModel) Update(msg tea.Msg) (DeleteModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	s := key.String()
	if s == "y" {
		if err := m.db.DeleteTrainer(m.trainer.ID); err != nil {
			return m, screen.ChangeScreen(screen.ScreenList, err.Error())
		}
		return m, screen.ChangeScreen(screen.ScreenList)
	}
	if s == "n" || s == "esc" {
		return m, screen.ChangeScreen(screen.ScreenList)
	}
	return m, nil
}

// View renders the delete confirmation prompt.
func (m DeleteModel) View() string {
	body := fmt.Sprintf("Delete %q?\n\n[y]es / [n]o", m.trainer.Label)
	return styles.RenderTitle("Delete trainer") + "\n\n" + body
}
