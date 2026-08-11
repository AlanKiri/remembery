package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/remembery/internal/beep"
	"github.com/alankiri/remembery/internal/engine"
	"github.com/alankiri/remembery/internal/store"
	"github.com/alankiri/remembery/internal/ui/common"
	"github.com/alankiri/remembery/internal/ui/screen"
	"github.com/alankiri/remembery/internal/ui/styles"
)

// TrainModel runs an active training session.
type TrainModel struct {
	db         *store.DB
	eng        *engine.Engine
	beeper     beep.Beep
	trainer    *store.Trainer
	counts     bool
	mask       engine.Mask
	input      string
	hint       bool
	attempt    int
	delaying   bool
	delayUntil time.Time
	countdown  int
	tStart     time.Time
	tErrors    int
}

// NewTrainModel creates a training model for the selected trainer.
func NewTrainModel(db *store.DB, eng *engine.Engine, beeper beep.Beep, t *store.Trainer, counts bool) (TrainModel, error) {
	m := TrainModel{
		db:      db,
		eng:     eng,
		beeper:  beeper,
		trainer: t,
		counts:  counts,
		tStart:  time.Now(),
	}
	mask, err := eng.MaskFor(t)
	if err != nil {
		return m, err
	}
	m.mask = mask
	return m, nil
}

// Init is a no-op for the training screen.
func (m TrainModel) Init() tea.Cmd {
	return nil
}

// Update handles key and timer messages during training.
func (m TrainModel) Update(msg tea.Msg) (TrainModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case common.TickMsg:
		return m.updateTick(msg)
	}
	return m, nil
}

func (m TrainModel) updateKey(msg tea.KeyMsg) (TrainModel, tea.Cmd) {
	if m.delaying {
		return m, nil
	}

	s := msg.String()

	switch s {
	case "esc":
		m.trainer = nil
		m.input = ""
		m.hint = false
		return m, screen.ChangeScreen(screen.ScreenList)
	case "ctrl+h":
		m.hint = !m.hint
		return m, nil
	case "backspace":
		r := []rune(m.input)
		if len(r) > 0 {
			m.input = string(r[:len(r)-1])
		}
		return m, nil
	case "enter":
		cmd := m.checkCorrect()
		if m.delaying {
			return m, tea.Batch(cmd, tea.Tick(time.Second, func(t time.Time) tea.Msg { return common.TickMsg{T: t} }))
		}
		return m, cmd
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
			cmd := m.checkCorrect()
			if m.delaying {
				return m, tea.Batch(cmd, tea.Tick(time.Second, func(t time.Time) tea.Msg { return common.TickMsg{T: t} }))
			}
			return m, cmd
		}
	}
	return m, nil
}

func (m *TrainModel) checkCorrect() tea.Cmd {
	if m.eng.Validate(m.input, string(m.mask.Password)) {
		return m.doCorrect()
	} else if len([]rune(m.input)) == len(m.mask.Password) {
		m.tErrors++
		m.input = ""
	}
	return nil
}

func (m *TrainModel) doCorrect() tea.Cmd {
	m.attempt++
	m.input = ""
	m.hint = false

	level := m.mask.Level
	if m.attempt >= level.RepetitionCount {
		return m.finishSession()
	}

	m.delaying = true
	m.delayUntil = time.Now().Add(time.Duration(level.InterAttemptDelay) * time.Second)
	m.countdown = level.InterAttemptDelay
	return nil
}

func (m *TrainModel) finishSession() tea.Cmd {
	var b strings.Builder
	b.WriteString("Great job!\n\n")

	nextScreen := screen.ScreenList
	var levelTrainer *store.Trainer
	levelOffer := 0

	if m.counts {
		now := time.Now()
		if err := m.eng.RecordSession(m.trainer, now, true, m.attempt, m.tErrors); err != nil {
			return screen.ChangeScreen(screen.ScreenList, err.Error())
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
			return screen.ChangeScreen(screen.ScreenList, err.Error())
		}
		if err := m.db.UpdateTrainer(*m.trainer); err != nil {
			return screen.ChangeScreen(screen.ScreenList, err.Error())
		}

		b.WriteString(fmt.Sprintf("Repetitions: %d\nErrors: %d\nTotal  sessions: %d\n",
			m.attempt, m.tErrors, m.trainer.TotalSessions))
		nextAvailable := now.Add(time.Duration(m.mask.Level.SessionIntervalHours) * time.Hour)
		b.WriteString(fmt.Sprintf("Next available in %s\n", common.FormatDuration(time.Until(nextAvailable))))

		can, next := m.eng.CanAdvance(m.trainer)
		if can {
			levelTrainer = m.trainer
			levelOffer = next
			nextScreen = screen.ScreenLevel
			b.WriteString(fmt.Sprintf("\nYou are ready to advance to level %d!", next))
		}
	} else {
		b.WriteString("Practice complete.\nNo progress was recorded because this trainer is not available yet.")
	}

	return common.ShowCongrats(b.String(), nextScreen, levelTrainer, levelOffer)
}

func (m TrainModel) updateTick(msg common.TickMsg) (TrainModel, tea.Cmd) {
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
			return m, screen.ChangeScreen(screen.ScreenList, err.Error())
		}
		m.mask = mask
		return m, nil
	}
	m.countdown = int(remaining.Seconds()) + 1
	return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return common.TickMsg{T: t} })
}

// View renders the training screen.
func (m TrainModel) View(w, h int) string {
	var b strings.Builder

	if m.trainer == nil || m.mask.Password == nil {
		return "loading..."
	}

	b.WriteString(styles.RenderTitle(fmt.Sprintf("Training: %s", m.trainer.Label)))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Level %d — attempt %d/%d",
		m.trainer.Level, m.attempt+1, m.mask.Level.RepetitionCount)))
	b.WriteString("\n\n")

	if m.delaying {
		msg := fmt.Sprintf("Correct! Next attempt in %d...\n\nA beep will be played once the timer ends.", m.countdown)
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Dim).Render(msg))
		return b.String()
	}

	promptText := m.mask.Blurred
	promptColor := styles.Glow.Fuchsia
	if m.hint {
		promptText = string(m.mask.Password)
		promptColor = styles.Glow.Yellow
	}

	inputRunes := []rune(m.input)
	pos := len(inputRunes)
	if pos >= len(m.mask.Password) {
		pos = len(m.mask.Password) - 1
	}
	if pos < 0 {
		pos = 0
	}
	leftPad := 1
	arrow := lipgloss.NewStyle().
		Foreground(styles.Glow.White).
		Render(strings.Repeat(" ", leftPad+pos) + "▼")

	b.WriteString(arrow)
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(promptColor).
		Padding(0, 0, 0, leftPad).
		Render(promptText))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Dim).Render("Input: " + m.input))
	b.WriteString("\n\n")

	status := fmt.Sprintf("typed: %d / %d", len([]rune(m.input)), len(m.mask.Password))
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Glow.Dim).Render(status))

	body := strings.TrimSuffix(b.String(), "\n")
	footer := styles.DimStyle.Render("type  Enter: submit  Backspace: remove  Ctrl+H: toggle hint  Esc: quit")

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
