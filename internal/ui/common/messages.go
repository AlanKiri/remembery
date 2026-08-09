package common

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alankiri/password-memorizer-tui/internal/store"
	"github.com/alankiri/password-memorizer-tui/internal/ui/screen"
)

// SetErrMsg sets the shared errMsg field on the main model.
type SetErrMsg struct{ Text string }

// SetErr returns a command that sends a SetErrMsg.
func SetErr(text string) tea.Cmd {
	return func() tea.Msg {
		return SetErrMsg{Text: text}
	}
}

// WelcomeTickMsg is sent by the welcome countdown timer.
type WelcomeTickMsg struct{ T time.Time }

// TickMsg is sent by the training inter-attempt timer.
type TickMsg struct{ T time.Time }

// StartEditMsg asks the main model to start editing a trainer.
type StartEditMsg struct{ Trainer *store.Trainer }

// StartEdit returns a command that sends a StartEditMsg.
func StartEdit(t *store.Trainer) tea.Cmd {
	return func() tea.Msg {
		return StartEditMsg{Trainer: t}
	}
}

// StartDeleteMsg asks the main model to show the delete confirmation for a trainer.
type StartDeleteMsg struct{ Trainer *store.Trainer }

// StartDelete returns a command that sends a StartDeleteMsg.
func StartDelete(t *store.Trainer) tea.Cmd {
	return func() tea.Msg {
		return StartDeleteMsg{Trainer: t}
	}
}

// ShowEarlyMsg asks the main model to show the early-train warning for a trainer.
type ShowEarlyMsg struct{ Trainer *store.Trainer }

// ShowEarly returns a command that sends a ShowEarlyMsg.
func ShowEarly(t *store.Trainer) tea.Cmd {
	return func() tea.Msg {
		return ShowEarlyMsg{Trainer: t}
	}
}

// StartTrainMsg asks the main model to begin training a trainer.
type StartTrainMsg struct {
	Trainer *store.Trainer
	Counts  bool
}

// StartTrain returns a command that sends a StartTrainMsg.
func StartTrain(t *store.Trainer, counts bool) tea.Cmd {
	return func() tea.Msg {
		return StartTrainMsg{Trainer: t, Counts: counts}
	}
}

// ShowCongratsMsg tells the main model to display the post-training summary.
type ShowCongratsMsg struct {
	Text         string
	NextScreen   screen.Screen
	LevelTrainer *store.Trainer
	LevelOffer   int
}

// ShowCongrats returns a command that sends a ShowCongratsMsg.
func ShowCongrats(text string, next screen.Screen, levelTrainer *store.Trainer, levelOffer int) tea.Cmd {
	return func() tea.Msg {
		return ShowCongratsMsg{
			Text:         text,
			NextScreen:   next,
			LevelTrainer: levelTrainer,
			LevelOffer:   levelOffer,
		}
	}
}
