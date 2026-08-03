package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/password-memorizer-tui/internal/store"
)

type tickMsg struct {
	t time.Time
}

func (m *Model) updateTrain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.delaying {
		return m, nil
	}

	s := msg.String()

	switch s {
	case "esc":
		m.screen = screenList
		m.trainer = nil
		m.input = ""
		m.hint = false
		return m, nil
	case "ctrl+h":
		m.hint = true
		m.input = ""
		return m, nil
	case "backspace":
		if m.hint {
			m.hint = false
			m.input = ""
			return m, nil
		}
		r := []rune(m.input)
		if len(r) > 0 {
			m.input = string(r[:len(r)-1])
		}
		return m, nil
	case "enter":
		if m.hint {
			m.hint = false
			m.input = ""
			return m, nil
		}
		m.checkCorrect()
		if m.delaying {
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{t} })
		}
		return m, nil
	}

	if m.hint {
		m.hint = false
		m.input = ""
	}

	if msg.Type == tea.KeyRunes {
		inputRunes := []rune(m.input)
		target := m.mask.Password
		if len(inputRunes) < len(target) {
			toAdd := msg.Runes
			max := len(target) - len(inputRunes)
			if len(toAdd) > max {
				toAdd = toAdd[:max]
			}
			m.input = string(append(inputRunes, toAdd...))
			m.checkCorrect()
			if m.delaying {
				return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{t} })
			}
		}
	}
	return m, nil
}

func (m *Model) checkCorrect() {
	if m.eng.Validate(m.input, string(m.mask.Password)) {
		m.doCorrect()
	} else if len([]rune(m.input)) == len(m.mask.Password) {
		m.tErrors++
		m.input = ""
	}
}

func (m *Model) doCorrect() {
	m.attempt++
	m.input = ""
	m.hint = false

	level := m.mask.Level
	if m.attempt >= level.RepetitionCount {
		m.finishSession()
		return
	}

	m.delaying = true
	m.delayUntil = time.Now().Add(time.Duration(level.InterAttemptDelay) * time.Second)
	m.countdown = level.InterAttemptDelay
}

func (m *Model) finishSession() {
	now := time.Now()
	if err := m.eng.RecordSession(m.trainer, m.tStart, true, m.attempt, m.tErrors); err != nil {
		m.errMsg = err.Error()
		m.screen = screenList
		return
	}

	sess := store.Session{
		TrainerID:   m.trainer.ID,
		StartedAt:   m.tStart,
		CompletedAt: &now,
		Repetitions: m.attempt,
		Errors:      m.tErrors,
		Successful:  true,
	}
	if _, err := m.db.CreateSession(sess); err != nil {
		m.errMsg = err.Error()
		m.screen = screenList
		return
	}
	if err := m.db.UpdateTrainer(*m.trainer); err != nil {
		m.errMsg = err.Error()
		m.screen = screenList
		return
	}

	can, next := m.eng.CanAdvance(m.trainer)
	if can {
		m.levelTrainer = m.trainer
		m.levelOffer = next
		m.screen = screenLevel
		return
	}

	m.loadTrainers()
	m.screen = screenList
}

func (m *Model) updateTrainTick(msg tickMsg) (tea.Model, tea.Cmd) {
	if !m.delaying {
		return m, nil
	}
	remaining := time.Until(m.delayUntil)
	if remaining <= 0 {
		m.delaying = false
		m.hint = false
		m.input = ""
		m.beeper.Beep()
		mask, err := m.eng.MaskFor(m.trainer)
		if err != nil {
			m.errMsg = err.Error()
			m.screen = screenList
			return m, nil
		}
		m.mask = mask
		return m, nil
	}
	m.countdown = int(remaining.Seconds()) + 1
	return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{t} })
}

func (m *Model) updateLevel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if s == "y" {
		m.levelTrainer.Level = m.levelOffer
		m.levelTrainer.SessionsAtLevel = 0
		if err := m.db.UpdateTrainer(*m.levelTrainer); err != nil {
			m.errMsg = err.Error()
		}
		m.loadTrainers()
		m.screen = screenList
	} else if s == "n" || s == "esc" {
		m.loadTrainers()
		m.screen = screenList
	}
	return m, nil
}

func (m Model) trainView() string {
	var b strings.Builder

	if m.trainer == nil {
		return "loading..."
	}

	b.WriteString(lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Training: %s (level %d, attempt %d/%d)",
		m.trainer.Label, m.trainer.Level, m.attempt+1, m.mask.Level.RepetitionCount)))
	b.WriteString("\n\n")

	if m.hint {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Render("HINT — full password"))
		b.WriteString("\n")
		b.WriteString(string(m.mask.Password))
		b.WriteString("\n\n")
		b.WriteString("Press any key to hide the hint and continue.")
		return b.String()
	}

	if m.delaying {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(fmt.Sprintf("Correct! Next attempt in %d...", m.countdown)))
		return b.String()
	}

	validation := m.eng.ValidateRunes([]rune(m.input), m.mask.Password)
	prompt := m.eng.Display(m.input, m.mask, validation)
	b.WriteString(prompt)
	b.WriteString("\n\n")

	status := fmt.Sprintf("typed: %d / %d", len([]rune(m.input)), len(m.mask.Password))
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(status))
	b.WriteString("\n\n")

	inst := []string{
		"Type the full password.",
		"Enter: submit",
		"Backspace: remove last character",
		"Ctrl+H: hint",
		"Esc: quit training",
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(strings.Join(inst, " • ")))
	return b.String()
}

func (m Model) levelView() string {
	return fmt.Sprintf("You are ready to advance %q to level %d.\n\nWarning: sessions at level will reset.\n\n[y]es / [n]o",
		m.levelTrainer.Label, m.levelOffer)
}
