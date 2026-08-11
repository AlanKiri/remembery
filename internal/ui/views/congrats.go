package views

import (
	"github.com/alankiri/remembery/internal/ui/screen"
	"github.com/alankiri/remembery/internal/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

// CongratsModel displays the post-training summary and advances to the next
// screen when any key is pressed.
type CongratsModel struct {
	text       string
	nextScreen screen.Screen
}

// NewCongratsModel creates a congrats model with the rendered summary text and
// the screen to switch to on the next key press.
func NewCongratsModel(text string, next screen.Screen) CongratsModel {
	return CongratsModel{
		text:       text,
		nextScreen: next,
	}
}

// Init is a no-op for the congrats screen.
func (m CongratsModel) Init() tea.Cmd {
	return nil
}

// Update advances to the next screen on any key press.
func (m CongratsModel) Update(msg tea.Msg) (CongratsModel, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		return m, screen.ChangeScreen(m.nextScreen)
	}
	return m, nil
}

// View renders the congrats screen.
func (m CongratsModel) View() string {
	return styles.RenderTitle("Session complete") + "\n\n" + m.text + "\n\nPress any key to continue."
}
