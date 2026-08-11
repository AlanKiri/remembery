package views

import (
	"errors"
	"fmt"
	"strings"
	"time"

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

// EditModel handles editing a trainer's label and level.
type EditModel struct {
	db         *store.DB
	eng        *engine.Engine
	levels     []levels.Level
	trainer    *store.Trainer
	inputs     [1]textinput.Model
	focus      int
	levelIndex int
}

// NewEditModel creates an edit model for the given trainer.
func NewEditModel(db *store.DB, eng *engine.Engine, levels []levels.Level, t *store.Trainer) EditModel {
	m := EditModel{
		db:         db,
		eng:        eng,
		levels:     levels,
		trainer:    t,
		levelIndex: common.FindLevelIndex(levels, t.Level),
		focus:      1,
	}
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Label"
	m.inputs[0].Prompt = "  "
	m.inputs[0].SetValue(t.Label)
	m.inputs[0].Blur()
	return m
}

// Init is a no-op for the edit trainer screen.
func (m EditModel) Init() tea.Cmd {
	return nil
}

// saveEdit persists the edited label and level.
func (m *EditModel) saveEdit() error {
	if m.trainer == nil {
		return errors.New("no trainer selected")
	}
	label := strings.TrimSpace(m.inputs[0].Value())
	if label == "" {
		return errors.New("label is required")
	}
	m.trainer.Label = label
	newLevel := m.levels[m.levelIndex].Number
	if m.trainer.Level != newLevel {
		m.trainer.Level = newLevel
		m.trainer.SessionsAtLevel = 0
		m.trainer.LastCountedSession = nil
		m.trainer.LastResetDate = time.Now()
	}
	return m.db.UpdateTrainer(*m.trainer)
}

// Update handles input and navigation when editing a trainer.
func (m EditModel) Update(msg tea.Msg) (EditModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	s := key.String()

	if s == "esc" {
		return m, screen.ChangeScreen(screen.ScreenList)
	}
	if s == "tab" || s == "shift+tab" {
		dir := 1
		if s == "shift+tab" {
			dir = -1
		}
		if m.focus < 1 {
			m.inputs[m.focus].Blur()
		}
		m.focus = (m.focus + dir + 2) % 2
		if m.focus < 1 {
			m.inputs[m.focus].Focus()
		}
		return m, nil
	}
	if m.focus == 1 {
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
			if err := m.saveEdit(); err != nil {
				return m, common.SetErr(err.Error())
			}
			return m, screen.ChangeScreen(screen.ScreenList)
		}
		return m, nil
	}
	if s == "enter" {
		if err := m.saveEdit(); err != nil {
			return m, common.SetErr(err.Error())
		}
		return m, screen.ChangeScreen(screen.ScreenList)
	}

	if m.focus == 0 {
		m.inputs[0], _ = m.inputs[0].Update(msg)
	}
	return m, nil
}

// View renders the edit trainer form.
func (m EditModel) View(w, h int, errMsg string) string {
	var b strings.Builder
	b.WriteString(styles.RenderTitle("Edit trainer"))
	b.WriteString("\n\n")

	style := lipgloss.NewStyle().Foreground(styles.Glow.Cream)
	if m.focus == 0 {
		style = lipgloss.NewStyle().Bold(true).Foreground(styles.Glow.Fuchsia)
	}
	b.WriteString(style.Render("  Label:"))
	b.WriteString("\n")
	b.WriteString(m.inputs[0].View())
	b.WriteString("\n\n")

	b.WriteString("  ")
	if m.focus == 1 {
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
	if m.trainer != nil {
		b.WriteString(common.BlurPreview(m.eng, m.levels, m.trainer.Password, m.levelIndex))
	}

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
