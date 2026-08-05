package engine

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/alankiri/password-memorizer-tui/internal/levels"
	"github.com/alankiri/password-memorizer-tui/internal/store"
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

func (e *Engine) Hint(mask Mask) (string, []rune) {
	return string(mask.Password), mask.Password
}

func (e *Engine) Validate(input, password string) bool {
	return input == password
}

func (e *Engine) ValidateRunes(input, password []rune) []bool {
	ok := make([]bool, len(input))
	for i := 0; i < len(input) && i < len(password); i++ {
		ok[i] = input[i] == password[i]
	}
	return ok
}

func (e *Engine) CanAdvance(trainer *store.Trainer) (bool, int) {
	level, ok := e.LevelConfig(trainer.Level)
	if !ok {
		return false, trainer.Level
	}
	if trainer.SessionsAtLevel < level.RequiredSessionsToProgress {
		return false, trainer.Level
	}
	next := trainer.Level + 1
	_, ok = e.LevelConfig(next)
	if !ok {
		return false, trainer.Level
	}
	return true, next
}

func (e *Engine) AdvanceIfReady(trainer *store.Trainer) (advanced bool) {
	can, next := e.CanAdvance(trainer)
	if !can {
		return false
	}
	trainer.Level = next
	trainer.SessionsAtLevel = 0
	return true
}

func (e *Engine) RecordSession(trainer *store.Trainer, start time.Time, successful bool, repetitions, errors int) error {
	level, ok := e.LevelConfig(trainer.Level)
	if !ok {
		return fmt.Errorf("unknown level %d", trainer.Level)
	}

	trainer.TotalSessions++
	if successful {
		trainer.SessionsAtLevel++
		next := start.Add(time.Duration(level.SessionIntervalDays) * 24 * time.Hour)
		trainer.NextDue = &next
	}
	return nil
}

func (e *Engine) HiddenPositions(mask Mask) []int {
	var out []int
	for i := range mask.Password {
		if mask.Hidden[i] {
			out = append(out, i)
		}
	}
	sort.Ints(out)
	return out
}

func (e *Engine) Display(input string, mask Mask, validation []bool) string {
	var parts []string
	ir := []rune(input)
	pr := mask.Password

	for i := 0; i < len(pr); i++ {
		var ch string
		if i < len(ir) {
			ch = string(ir[i])
			if len(validation) > i && !validation[i] {
				ch = "[" + ch + "]"
			}
		} else if mask.Hidden[i] {
			ch = "•"
		} else {
			ch = string(pr[i])
		}
		parts = append(parts, ch)
	}
	return strings.Join(parts, " ")
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
