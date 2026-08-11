package screen

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Screen identifies one of the application's top-level screens.
type Screen int

const (
	ScreenWelcome Screen = iota
	ScreenList
	ScreenNew
	ScreenEdit
	ScreenDelete
	ScreenEarlyTrain
	ScreenTrain
	ScreenCongrats
	ScreenLevel
	ScreenVault
)

// ChangeScreenMsg requests the main model to switch to a different screen.
type ChangeScreenMsg struct {
	To  Screen
	Err string
}

// ChangeScreen returns a command that sends a ChangeScreenMsg. An optional
// error is exposed as the main model's errMsg when the screen changes.
func ChangeScreen(s Screen, err ...string) tea.Cmd {
	var e string
	if len(err) > 0 {
		e = err[0]
	}
	return func() tea.Msg {
		return ChangeScreenMsg{To: s, Err: e}
	}
}
