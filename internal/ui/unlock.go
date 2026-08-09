package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/password-memorizer-tui/internal/store"
	"github.com/alankiri/password-memorizer-tui/internal/ui/styles"
	"github.com/alankiri/password-memorizer-tui/internal/vault"
)

type unlockState int

const (
	stateChoice unlockState = iota
	statePassword
	stateConfirm
	stateResetWarn
)

type unlockModel struct {
	db       *store.DB
	creating bool
	state    unlockState
	pw       textinput.Model
	confirm  textinput.Model
	errMsg   string
	result   *vault.Vault
	skipped  bool
	reset    bool
	showPw   bool
	width    int
	height   int
}

func newUnlockModel(db *store.DB, creating bool) unlockModel {
	pw := textinput.New()
	pw.Placeholder = "master password"
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.CharLimit = 64
	pw.Focus()

	confirm := textinput.New()
	confirm.Placeholder = "confirm password"
	confirm.EchoMode = textinput.EchoPassword
	confirm.EchoCharacter = '•'
	confirm.CharLimit = 64

	st := statePassword
	if creating {
		st = stateChoice
	}

	return unlockModel{
		db:       db,
		creating: creating,
		state:    st,
		pw:       pw,
		confirm:  confirm,
		showPw:   false,
	}
}

func (m unlockModel) Init() tea.Cmd {
	return m.pw.Focus()
}

func (m *unlockModel) applyEcho() {
	if m.showPw {
		m.pw.EchoMode = textinput.EchoNormal
		m.confirm.EchoMode = textinput.EchoNormal
	} else {
		m.pw.EchoMode = textinput.EchoPassword
		m.confirm.EchoMode = textinput.EchoPassword
	}
	m.pw.EchoCharacter = '•'
	m.confirm.EchoCharacter = '•'
}

func (m unlockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		s := msg.String()
		if s == "q" {
			return m, tea.Quit
		}
		if s == "y" && m.state == stateChoice {
			m.state = statePassword
			m.pw.Focus()
			m.errMsg = ""
			return m, nil
		}
		if s == "n" && m.state == stateChoice {
			m.skipped = true
			return m, tea.Quit
		}
		if s == "r" && m.state == statePassword && !m.creating {
			m.state = stateResetWarn
			m.errMsg = ""
			return m, nil
		}
		if (m.state == statePassword || m.state == stateConfirm) && s == "ctrl+s" {
			m.showPw = !m.showPw
			m.applyEcho()
			return m, nil
		}
		if m.state == stateResetWarn {
			if s == "n" {
				m.state = statePassword
				m.errMsg = ""
				return m, nil
			}
			if s == "y" {
				if err := m.db.Reset(); err != nil {
					m.errMsg = err.Error()
					return m, nil
				}
				m.reset = true
				return m, tea.Quit
			}
			return m, nil
		}
		if s == "enter" {
			return m.advance()
		}
	}

	var cmd tea.Cmd
	if m.state == stateConfirm {
		m.confirm, cmd = m.confirm.Update(msg)
	} else {
		m.pw, cmd = m.pw.Update(msg)
	}
	return m, cmd
}

func (m unlockModel) advance() (tea.Model, tea.Cmd) {
	m.errMsg = ""
	switch m.state {
	case stateChoice:
		// Should not happen; y/n handles the choice.
		return m, nil
	case statePassword:
		password := m.pw.Value()
		if password == "" {
			m.errMsg = "Password cannot be empty"
			return m, nil
		}
		if !m.creating {
			v, err := m.db.LoadVault(password)
			if err != nil {
				m.errMsg = "Incorrect master password"
				return m, nil
			}
			m.result = v
			return m, tea.Quit
		}
		m.state = stateConfirm
		m.confirm.Focus()
		return m, nil
	case stateConfirm:
		password := m.pw.Value()
		if m.confirm.Value() != password {
			m.errMsg = "Passwords do not match"
			return m, nil
		}
		v, err := m.db.CreateVault(password)
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.result = v
		return m, tea.Quit
	}
	return m, nil
}

func (m unlockModel) View() string {
	var body strings.Builder

	switch m.state {
	case stateChoice:
		body.WriteString("Protect your stored passwords with a master key?\n\n")
		body.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Green).Render("[y]es"))
		body.WriteString(" — enable encryption\n")
		body.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Red).Render("[n]o"))
		body.WriteString(" — continue without encryption\n")
	case statePassword:
		if m.creating {
			body.WriteString("Create a master password for your vault.\n")
			body.WriteString(styles.DimStyle.Render("If you forget it, your stored passwords cannot be recovered."))
			body.WriteString("\n\n")
		} else {
			body.WriteString("Enter your master password to unlock the vault.\n\n")
		}
		body.WriteString(m.pw.View())
		body.WriteString("\n")
	case stateConfirm:
		body.WriteString("Re-enter the same master password to confirm.\n\n")
		body.WriteString(m.confirm.View())
		body.WriteString("\n")
	case stateResetWarn:
		body.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Red).Bold(true).Render("WARNING: this will erase all stored data."))
		body.WriteString("\n\n")
		body.WriteString("If you forgot your master password, resetting is the only way to start over.\n\n")
		body.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Green).Render("[y]es"))
		body.WriteString(" to confirm • ")
		body.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Red).Render("[n]o"))
		body.WriteString(" to cancel")
		body.WriteString("\n")
	}

	if m.errMsg != "" {
		body.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Red).Render(m.errMsg))
		body.WriteString("\n")
	}

	var footerText string
	switch m.state {
	case stateChoice:
		footerText = "y: enable • n: skip • q: quit"
	case statePassword:
		if m.creating {
			footerText = "enter: continue • ctrl+s: show • q: quit"
		} else {
			footerText = styles.DimStyle.Render("enter: unlock • ctrl+s: show • ") +
				lipgloss.NewStyle().Foreground(styles.Glow.Red).Render("r: reset") +
				styles.DimStyle.Render(" • q: quit")
		}
	case stateConfirm:
		footerText = "enter: create • ctrl+s: show • q: quit"
	case stateResetWarn:
		footerText = "y: confirm • n: cancel • q: quit"
	}
	footer := styles.DimStyle.Padding(0, 2).Render(footerText)

	header := lipgloss.NewStyle().Padding(0, 2).Render(styles.RenderTitle("Master password"))
	info := lipgloss.NewStyle().Padding(0, 2).Render(body.String())
	content := "\n" + header + "\n\n" + info
	bodyLines := strings.Count(content, "\n") + 1
	footerLines := strings.Count(footer, "\n") + 1

	if m.height > 0 {
		gap := m.height - bodyLines - footerLines
		if gap < 0 {
			gap = 0
		}
		content += strings.Repeat("\n", gap) + footer
	} else {
		content += "\n\n" + footer
	}
	return content
}
