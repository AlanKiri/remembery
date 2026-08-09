package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alankiri/password-memorizer-tui/internal/engine"
	"github.com/alankiri/password-memorizer-tui/internal/store"
	"github.com/alankiri/password-memorizer-tui/internal/ui/screen"
	"github.com/alankiri/password-memorizer-tui/internal/ui/styles"
)

// LevelModel handles accepting or rejecting a level-up offer.
type LevelModel struct {
	db      *store.DB
	eng     *engine.Engine
	trainer *store.Trainer
	offer   int
}

// NewLevelModel creates a level-up offer model.
func NewLevelModel(db *store.DB, eng *engine.Engine, t *store.Trainer, offer int) LevelModel {
	return LevelModel{
		db:      db,
		eng:     eng,
		trainer: t,
		offer:   offer,
	}
}

// Init is a no-op for the level offer screen.
func (m LevelModel) Init() tea.Cmd {
	return nil
}

// Update handles accepting or declining the level-up offer.
func (m LevelModel) Update(msg tea.Msg) (LevelModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	s := key.String()
	if s == "y" {
		m.eng.AdvanceIfReady(m.trainer)
		if err := m.db.UpdateTrainer(*m.trainer); err != nil {
			return m, screen.ChangeScreen(screen.ScreenList, err.Error())
		}
		return m, screen.ChangeScreen(screen.ScreenList)
	}
	if s == "n" || s == "esc" {
		return m, screen.ChangeScreen(screen.ScreenList)
	}
	return m, nil
}

// View renders the level-up offer.
func (m LevelModel) View() string {
	body := fmt.Sprintf("You are ready to advance %q to level %d.\n\nWarning: sessions at level will reset.\n\n[y]es / [n]o",
		m.trainer.Label, m.offer)
	return styles.RenderTitle("Level up") + "\n\n" + body
}
