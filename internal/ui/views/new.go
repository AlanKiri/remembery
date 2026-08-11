package views

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/remembery/internal/engine"
	"github.com/alankiri/remembery/internal/levels"
	"github.com/alankiri/remembery/internal/store"
	"github.com/alankiri/remembery/internal/ui/common"
	"github.com/alankiri/remembery/internal/ui/screen"
	"github.com/alankiri/remembery/internal/ui/styles"
)

// NewModel handles creating a new trainer.
type NewModel struct {
	db           *store.DB
	eng          *engine.Engine
	levels       []levels.Level
	inputs       [2]textinput.Model
	focus        int
	levelIndex   int
	showPassword bool
}

// NewNewModel creates a new trainer creation model in its initial state.
func NewNewModel(db *store.DB, eng *engine.Engine, levels []levels.Level) NewModel {
	m := NewModel{
		db:     db,
		eng:    eng,
		levels: levels,
	}
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Label"
	m.inputs[0].Prompt = "  "
	m.inputs[0].Focus()
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Password"
	m.inputs[1].EchoMode = textinput.EchoPassword
	m.inputs[1].Prompt = "  "
	m.focus = 0
	m.levelIndex = 0
	return m
}

// Init is a no-op for the new trainer screen.
func (m NewModel) Init() tea.Cmd {
	return nil
}

// createTrainer persists a new trainer from the input values.
func (m NewModel) createTrainer() error {
	label := strings.TrimSpace(m.inputs[0].Value())
	password := m.inputs[1].Value()
	if label == "" {
		return errors.New("label is required")
	}
	if password == "" {
		return errors.New("password is required")
	}
	level := m.levels[m.levelIndex].Number
	_, err := m.db.CreateTrainer(label, password, level)
	return err
}

// Update handles input and navigation when creating a new trainer.
func (m NewModel) Update(msg tea.Msg) (NewModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	s := key.String()

	if s == "esc" {
		return m, screen.ChangeScreen(screen.ScreenList)
	}
	if s == "ctrl+s" {
		m.showPassword = !m.showPassword
		if m.showPassword {
			m.inputs[1].EchoMode = textinput.EchoNormal
		} else {
			m.inputs[1].EchoMode = textinput.EchoPassword
		}
		return m, nil
	}
	if s == "tab" || s == "shift+tab" {
		dir := 1
		if s == "shift+tab" {
			dir = -1
		}
		if m.focus < 2 {
			m.inputs[m.focus].Blur()
		}
		m.focus = (m.focus + dir + 3) % 3
		if m.focus < 2 {
			m.inputs[m.focus].Focus()
		}
		return m, nil
	}
	if m.focus == 2 {
		switch s {
		case "k", "up":
			if m.levelIndex > 0 {
				m.levelIndex--
			}
		case "j", "down":
			if m.levelIndex < len(m.levels)-1 {
				m.levelIndex++
			}
		case "enter":
			if err := m.createTrainer(); err != nil {
				return m, common.SetErr(err.Error())
			}
			return m, screen.ChangeScreen(screen.ScreenList)
		}
		return m, nil
	}

	if s == "enter" {
		if err := m.createTrainer(); err != nil {
			return m, common.SetErr(err.Error())
		}
		return m, screen.ChangeScreen(screen.ScreenList)
	}

	i := m.focus
	if i >= 0 && i < 2 {
		m.inputs[i], _ = m.inputs[i].Update(msg)
	}
	return m, nil
}

// View renders the new trainer form.
func (m NewModel) View(w, h int, errMsg string) string {
	var b strings.Builder
	b.WriteString(styles.RenderTitle("New trainer"))
	b.WriteString("\n\n")

	for i := range m.inputs {
		name := []string{"Label", "Password"}[i]
		style := lipgloss.NewStyle().Foreground(styles.Glow.Cream)
		if i == m.focus {
			style = lipgloss.NewStyle().Bold(true).Foreground(styles.Glow.Fuchsia)
		}
		b.WriteString(style.Render("  " + name + ":"))
		b.WriteString("\n")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n")
	}

	check := "[ ]"
	if m.showPassword {
		check = "[x]"
	}
	b.WriteString(styles.DimStyle.Render("  " + check + " Show password (ctrl+s to toggle)"))
	b.WriteString("\n\n")

	b.WriteString("  ")
	if m.focus == 2 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.Glow.Fuchsia).Render("Familiarity level"))
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Dim).Render(" (j/k to select)"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Cream).Render("Familiarity level"))
	}
	b.WriteString("\n")
	descWidth := w - 26
	if descWidth < 20 {
		descWidth = 20
	}
	for i, l := range m.levels {
		var left string
		if i == m.levelIndex {
			left = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color(l.Color)).
				Foreground(lipgloss.Color("#fff")).
				Width(18).
				Render(fmt.Sprintf("  Level %d", l.Number))
		} else {
			left = lipgloss.NewStyle().
				Foreground(lipgloss.Color(l.Color)).
				Width(18).
				Render(fmt.Sprintf("  Level %d", l.Number))
		}
		if i == m.levelIndex {
			right := lipgloss.NewStyle().
				Foreground(styles.Glow.Dim).
				Width(descWidth).
				Render(l.Description)
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))
		} else {
			b.WriteString(left)
		}
		b.WriteString("\n")
	}
	b.WriteString(common.BlurPreview(m.eng, m.levels, m.inputs[1].Value(), m.levelIndex))

	if errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Red).Render("error: " + errMsg))
	}

	body := strings.TrimSuffix(b.String(), "\n")
	footer := styles.DimStyle.Render("tab: switch focus  enter: save  esc: cancel")

	if h > 0 {
		bodyLines := strings.Count(body, "\n") + 1
		footerLines := strings.Count(footer, "\n") + 1
		gap := h - bodyLines - footerLines
		if gap < 0 {
			gap = 0
		}
		body += strings.Repeat("\n", gap) + footer
	} else {
		body += "\n\n" + footer
	}
	return body
}
