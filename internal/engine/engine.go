package engine

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/alankiri/remembery/internal/levels"
	"github.com/alankiri/remembery/internal/store"
)

type Engine struct {
	Levels []levels.Level
}

func New(levels []levels.Level) *Engine {
	return &Engine{Levels: levels}
}

func (e *Engine) LevelConfig(n int) (levels.Level, bool) {
	for _, l := range e.Levels {
		if l.Number == n {
			return l, true
		}
	}
	return levels.Level{}, false
}

type Mask struct {
	Hidden   map[int]bool
	Revealed []rune
	Blurred  string
	Level    levels.Level
	Password []rune
}

func (e *Engine) MaskFor(trainer *store.Trainer) (Mask, error) {
	level, ok := e.LevelConfig(trainer.Level)
	if !ok {
		return Mask{}, fmt.Errorf("unknown level %d", trainer.Level)
	}
	return e.maskForPassword(trainer.Password, level, trainer.SessionsAtLevel)
}

func (e *Engine) Preview(level levels.Level, password string) (Mask, error) {
	return e.maskForPassword(password, level, 0)
}

func (e *Engine) maskForPassword(password string, level levels.Level, sessionsAtLevel int) (Mask, error) {
	runes := []rune(password)
	n := len(runes)
	if n == 0 {
		return Mask{}, fmt.Errorf("password cannot be empty")
	}

	effective := clamp(level.BaseBlurPercent+sessionsAtLevel*level.BlurStepPercent, 0, level.MaxBlurPercent)
	hiddenCount := int(math.Ceil(float64(n) * float64(effective) / 100.0))
	if hiddenCount > n-1 {
		hiddenCount = n - 1
	}
	if hiddenCount < 0 {
		hiddenCount = 0
	}

	hidden := make(map[int]bool, hiddenCount)
	if hiddenCount > 0 {
		indices := rand.Perm(n)[:hiddenCount]
		for _, i := range indices {
			hidden[i] = true
		}
	}

	revealed := make([]rune, n)
	copy(revealed, runes)
	for i := range revealed {
		if hidden[i] {
			revealed[i] = '•'
		}
	}

	return Mask{
		Hidden:   hidden,
		Revealed: revealed,
		Blurred:  string(revealed),
		Level:    level,
		Password: runes,
	}, nil
}

func (e *Engine) Validate(input, password string) bool {
	return input == password
}

func (e *Engine) CanAdvance(trainer *store.Trainer) (bool, int) {
	level, ok := e.LevelConfig(trainer.Level)
	if !ok {
		return false, trainer.Level
	}
	if trainer.SessionsAtLevel < level.RequiredSessionsToProgress {
		return false, trainer.Level
	}

	var next int
	found := false
	for _, l := range e.Levels {
		if l.Number > trainer.Level {
			if !found || l.Number < next {
				next = l.Number
				found = true
			}
		}
	}
	if !found {
		return false, trainer.Level
	}
	return true, next
}

const graceWindow = 6 * time.Hour

func (e *Engine) AdvanceIfReady(trainer *store.Trainer) (advanced bool) {
	can, next := e.CanAdvance(trainer)
	if !can {
		return false
	}
	trainer.Level = next
	trainer.SessionsAtLevel = 0
	trainer.LastCountedSession = nil
	trainer.LastResetDate = time.Now()
	return true
}

func (e *Engine) RecordSession(trainer *store.Trainer, completedAt time.Time, successful bool, repetitions, errors int) error {
	if _, ok := e.LevelConfig(trainer.Level); !ok {
		return fmt.Errorf("unknown level %d", trainer.Level)
	}

	trainer.TotalSessions++
	if successful {
		trainer.SessionsAtLevel++
		trainer.LastCountedSession = &completedAt
	}
	return nil
}

// Schedule returns the current schedule for a trainer.
// A newly created or reset trainer is available immediately; the interval
// only applies after a counted session has been recorded.
func (e *Engine) Schedule(t store.Trainer) (availableAt, dueAt time.Time, status string, canCount bool) {
	level, ok := e.LevelConfig(t.Level)
	if !ok {
		status = "unknown"
		return
	}

	now := time.Now()
	if t.LastCountedSession != nil {
		availableAt = t.LastCountedSession.Add(time.Duration(level.SessionIntervalHours) * time.Hour)
	} else {
		availableAt = t.LastResetDate
	}
	dueAt = availableAt.Add(graceWindow)

	switch {
	case now.Before(availableAt):
		return availableAt, dueAt, "resting", false
	case now.Before(dueAt):
		return availableAt, dueAt, "available", true
	default:
		return availableAt, dueAt, "due", true
	}
}

// Availability returns the current schedule state for a trainer.
// canCount is true during the available and due windows.
func (e *Engine) Availability(t store.Trainer) (status string, canCount bool) {
	_, _, status, canCount = e.Schedule(t)
	return
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
