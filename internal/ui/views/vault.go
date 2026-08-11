package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/password-memorizer-tui/internal/config"
	"github.com/alankiri/password-memorizer-tui/internal/consts"
	"github.com/alankiri/password-memorizer-tui/internal/engine"
	"github.com/alankiri/password-memorizer-tui/internal/levels"
	"github.com/alankiri/password-memorizer-tui/internal/store"
	"github.com/alankiri/password-memorizer-tui/internal/ui/screen"
	"github.com/alankiri/password-memorizer-tui/internal/ui/styles"
)

type vaultStep int

const (
	vaultStepWarn vaultStep = iota
	vaultStepPassword
	vaultStepConfirm
	vaultStepMenu
	vaultStepChangePassword
	vaultStepChangeConfirm
	vaultStepDecryptWarn
)

// VaultModel is the child model for the vault screen.
type VaultModel struct {
	db           *store.DB
	cfg          *config.Config
	vaultStep    vaultStep
	vaultPw      textinput.Model
	vaultConfirm textinput.Model
	vaultShowPw  bool
	vaultErrMsg  string
	vaultFocus   int
}

// NewVaultModel creates a vault model in its initial state.
func NewVaultModel(db *store.DB, cfg *config.Config, _ *engine.Engine, _ []levels.Level) VaultModel {
	m := VaultModel{
		db:  db,
		cfg: cfg,
	}
	m.vaultPw = textinput.New()
	m.vaultPw.Placeholder = "master password"
	m.vaultPw.EchoMode = textinput.EchoPassword
	m.vaultPw.EchoCharacter = '•'
	m.vaultPw.CharLimit = 64

	m.vaultConfirm = textinput.New()
	m.vaultConfirm.Placeholder = "confirm password"
	m.vaultConfirm.EchoMode = textinput.EchoPassword
	m.vaultConfirm.EchoCharacter = '•'
	m.vaultConfirm.CharLimit = 64

	m.resetVault()
	return m
}

// Init is a no-op for the vault screen.
func (m VaultModel) Init() tea.Cmd {
	return nil
}

// View renders the vault screen.
func (m *VaultModel) View(w, h int) string {
	return m.vaultView()
}

// resetVault puts the vault screen into its initial vault state.
func (m *VaultModel) resetVault() {
	m.resetEnableVault()
	m.vaultFocus = 0
}

// resetEnableVault clears the vault password inputs and chooses the right
// starting step based on whether a vault already exists.
func (m *VaultModel) resetEnableVault() {
	m.vaultPw.SetValue("")
	m.vaultConfirm.SetValue("")
	m.vaultPw.EchoMode = textinput.EchoPassword
	m.vaultPw.EchoCharacter = '•'
	m.vaultConfirm.EchoMode = textinput.EchoPassword
	m.vaultConfirm.EchoCharacter = '•'
	m.vaultShowPw = false
	m.vaultErrMsg = ""
	if m.db.HasVault() {
		m.vaultStep = vaultStepMenu
	} else {
		m.vaultStep = vaultStepWarn
		m.vaultPw.Focus()
		m.vaultConfirm.Blur()
	}
}

// applyVaultEcho toggles the password inputs between masked and visible
// based on m.vaultShowPw.
func (m *VaultModel) applyVaultEcho() {
	if m.vaultShowPw {
		m.vaultPw.EchoMode = textinput.EchoNormal
		m.vaultConfirm.EchoMode = textinput.EchoNormal
	} else {
		m.vaultPw.EchoMode = textinput.EchoPassword
		m.vaultConfirm.EchoMode = textinput.EchoPassword
	}
	m.vaultPw.EchoCharacter = '•'
	m.vaultConfirm.EchoCharacter = '•'
}

// isVaultInputStep reports whether the vault is currently waiting for a
// master-password text input.
func (m VaultModel) isVaultInputStep() bool {
	switch m.vaultStep {
	case vaultStepPassword, vaultStepConfirm, vaultStepChangePassword, vaultStepChangeConfirm:
		return true
	}
	return false
}

// clampFocus keeps the focus index inside the current action list.
func (m *VaultModel) clampFocus(actions []string) {
	if len(actions) == 0 {
		m.vaultFocus = 0
		return
	}
	if m.vaultFocus < 0 {
		m.vaultFocus = 0
	}
	if m.vaultFocus >= len(actions) {
		m.vaultFocus = len(actions) - 1
	}
}

// Update routes input for the vault screen.
func (m *VaultModel) Update(msg tea.Msg) (VaultModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return *m, nil
	}
	s := key.String()

	// When a vault password prompt is active, all input goes to that prompt.
	if m.isVaultInputStep() {
		return m.updateVaultInput(key)
	}

	if s == "q" || s == "esc" {
		return *m, screen.ChangeScreen(screen.ScreenList)
	}

	actions := m.actionLabels()
	m.clampFocus(actions)
	if len(actions) == 0 {
		return *m, nil
	}

	switch s {
	case "j", "down":
		if m.vaultFocus < len(actions)-1 {
			m.vaultFocus++
		}
	case "k", "up":
		if m.vaultFocus > 0 {
			m.vaultFocus--
		}
	case "enter":
		return m.handleVaultAction(actions[m.vaultFocus])
	}

	return *m, nil
}

// updateVaultInput reads keys while a vault password prompt is open.
func (m *VaultModel) updateVaultInput(msg tea.KeyMsg) (VaultModel, tea.Cmd) {
	s := msg.String()
	if s == "q" {
		return *m, screen.ChangeScreen(screen.ScreenList)
	}
	if s == "esc" {
		return m.cancelVaultInput()
	}
	if s == "ctrl+s" {
		m.vaultShowPw = !m.vaultShowPw
		m.applyVaultEcho()
		return *m, nil
	}
	if s == "enter" {
		return m.submitVaultInput()
	}
	var cmd tea.Cmd
	switch m.vaultStep {
	case vaultStepPassword, vaultStepChangePassword:
		m.vaultPw, cmd = m.vaultPw.Update(msg)
	case vaultStepConfirm, vaultStepChangeConfirm:
		m.vaultConfirm, cmd = m.vaultConfirm.Update(msg)
	}
	return *m, cmd
}

// cancelVaultInput backs out of the current vault password prompt.
func (m *VaultModel) cancelVaultInput() (VaultModel, tea.Cmd) {
	m.vaultErrMsg = ""
	switch m.vaultStep {
	case vaultStepPassword:
		m.vaultStep = vaultStepWarn
		m.vaultPw.SetValue("")
		m.vaultConfirm.SetValue("")
		m.vaultPw.Blur()
		m.vaultConfirm.Blur()
	case vaultStepConfirm:
		m.vaultStep = vaultStepPassword
		m.vaultConfirm.SetValue("")
		m.vaultPw.Focus()
		m.vaultConfirm.Blur()
	case vaultStepChangePassword:
		m.vaultStep = vaultStepMenu
		m.vaultPw.SetValue("")
		m.vaultConfirm.SetValue("")
		m.vaultPw.Blur()
		m.vaultConfirm.Blur()
	case vaultStepChangeConfirm:
		m.vaultStep = vaultStepChangePassword
		m.vaultConfirm.SetValue("")
		m.vaultPw.Focus()
		m.vaultConfirm.Blur()
	}
	return *m, nil
}

// submitVaultInput confirms the active vault password prompt.
func (m *VaultModel) submitVaultInput() (VaultModel, tea.Cmd) {
	m.vaultErrMsg = ""
	switch m.vaultStep {
	case vaultStepPassword:
		m.vaultStep = vaultStepConfirm
		m.vaultConfirm.SetValue("")
		m.vaultConfirm.Focus()
		m.vaultPw.Blur()
		return *m, nil
	case vaultStepConfirm:
		if m.vaultPw.Value() != m.vaultConfirm.Value() {
			m.vaultErrMsg = "Passwords do not match"
			return *m, nil
		}
		v, err := m.db.CreateVaultAndEncrypt(m.vaultPw.Value())
		if err != nil {
			m.vaultErrMsg = err.Error()
			return *m, nil
		}
		m.db.SetVault(v)
		m.cfg.PromptedForVault = true
		if err := config.Save(*m.cfg); err != nil {
			m.vaultErrMsg = err.Error()
			return *m, nil
		}
		return *m, screen.ChangeScreen(screen.ScreenList)
	case vaultStepChangePassword:
		m.vaultStep = vaultStepChangeConfirm
		m.vaultConfirm.SetValue("")
		m.vaultConfirm.Focus()
		m.vaultPw.Blur()
		return *m, nil
	case vaultStepChangeConfirm:
		if m.vaultPw.Value() != m.vaultConfirm.Value() {
			m.vaultErrMsg = "Passwords do not match"
			return *m, nil
		}
		v, err := m.db.ChangeVault(m.vaultPw.Value())
		if err != nil {
			m.vaultErrMsg = err.Error()
			return *m, nil
		}
		m.db.SetVault(v)
		return *m, screen.ChangeScreen(screen.ScreenList)
	}
	return *m, nil
}

// actionLabels returns the selectable action labels for the current vault step.
func (m VaultModel) actionLabels() []string {
	switch m.vaultStep {
	case vaultStepWarn:
		return []string{"Enable vault"}
	case vaultStepMenu:
		return []string{"Change master password", "Remove vault"}
	case vaultStepDecryptWarn:
		return []string{"Remove vault"}
	}
	return nil
}

// handleVaultAction executes the selected vault action row.
func (m *VaultModel) handleVaultAction(label string) (VaultModel, tea.Cmd) {
	m.vaultErrMsg = ""
	switch m.vaultStep {
	case vaultStepWarn:
		m.vaultStep = vaultStepPassword
		m.vaultPw.SetValue("")
		m.vaultConfirm.SetValue("")
		m.vaultPw.Focus()
		m.vaultConfirm.Blur()
		m.vaultFocus = 0
	case vaultStepMenu:
		switch label {
		case "Change master password":
			m.vaultStep = vaultStepChangePassword
			m.vaultPw.SetValue("")
			m.vaultConfirm.SetValue("")
			m.vaultPw.Focus()
			m.vaultConfirm.Blur()
			m.vaultFocus = 0
		case "Remove vault":
			m.vaultStep = vaultStepDecryptWarn
			m.vaultFocus = 0
		}
	case vaultStepDecryptWarn:
		if err := m.db.DecryptVault(); err != nil {
			m.vaultErrMsg = err.Error()
			return *m, nil
		}
		return *m, screen.ChangeScreen(screen.ScreenList)
	}
	return *m, nil
}

// vaultView renders the vault screen.
func (m VaultModel) vaultView() string {
	var body strings.Builder
	body.WriteString(styles.RenderTitle("Vault"))
	body.WriteString("\n\n")

	switch m.vaultStep {
	case vaultStepWarn:
		body.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Red).Bold(true).Render("WARNING: this will encrypt all stored trainer passwords."))
		body.WriteString("\n")
		body.WriteString(fmt.Sprintf("Once enabled, you will need the master password every time you start %s.\n", consts.AppName))
		body.WriteString("If you forget it, there is no way to recover your stored passwords.\n")
	case vaultStepDecryptWarn:
		body.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Red).Bold(true).Render("WARNING: this will decrypt all stored passwords."))
		body.WriteString("\n")
		body.WriteString("The vault will be removed and you will no longer need a master password.\n")
	}
	if m.vaultStep == vaultStepWarn || m.vaultStep == vaultStepDecryptWarn {
		body.WriteString("\n")
	}

	if m.isVaultInputStep() {
		body.WriteString(m.renderVaultInputRow())
		body.WriteString("\n")
	} else {
		actions := m.actionLabels()
		m.clampFocus(actions)
		for i, label := range actions {
			body.WriteString(m.renderVaultActionRow(label, i == m.vaultFocus))
			body.WriteString("\n")
		}
	}

	if m.vaultErrMsg != "" {
		body.WriteString("\n")
		body.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Red).Render(m.vaultErrMsg))
		body.WriteString("\n")
	}

	var guide string
	switch m.vaultStep {
	case vaultStepWarn, vaultStepDecryptWarn:
		guide = "enter: confirm  esc: back"
	case vaultStepMenu:
		guide = "j/k: choose  enter: confirm  esc: back"
	case vaultStepPassword, vaultStepConfirm, vaultStepChangePassword, vaultStepChangeConfirm:
		guide = "enter: continue  ctrl+s: show  esc: back"
	}
	if guide != "" {
		body.WriteString("\n")
		body.WriteString(styles.DimStyle.Render(guide))
	}

	return body.String()
}

// renderVaultActionRow renders one selectable vault action.
func (m VaultModel) renderVaultActionRow(label string, focused bool) string {
	style := lipgloss.NewStyle().Foreground(styles.Glow.Cream)
	if focused {
		style = lipgloss.NewStyle().Bold(true).Foreground(styles.Glow.Blue)
	}
	return style.Render("  " + label)
}

// renderVaultInputRow renders the active vault password input.
func (m VaultModel) renderVaultInputRow() string {
	var label string
	var input textinput.Model
	switch m.vaultStep {
	case vaultStepPassword:
		label = "Master password: "
		input = m.vaultPw
	case vaultStepConfirm:
		label = "Confirm master password: "
		input = m.vaultConfirm
	case vaultStepChangePassword:
		label = "New master password: "
		input = m.vaultPw
	case vaultStepChangeConfirm:
		label = "Confirm new master password: "
		input = m.vaultConfirm
	}
	return lipgloss.NewStyle().Foreground(styles.Glow.Cream).Render("  "+label) + input.View()
}
