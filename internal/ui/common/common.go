package common

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/remembery/internal/engine"
	"github.com/alankiri/remembery/internal/levels"
	"github.com/alankiri/remembery/internal/ui/styles"
)

// FormatDuration formats a duration as hours and minutes.
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// LevelColor returns the color for the given level number.
func LevelColor(levels []levels.Level, n int) string {
	for _, l := range levels {
		if l.Number == n {
			return l.Color
		}
	}
	return "#fff"
}

// FindLevelIndex returns the index of the given level number.
func FindLevelIndex(levels []levels.Level, n int) int {
	for i, l := range levels {
		if l.Number == n {
			return i
		}
	}
	return 0
}

// BlurPreview renders a preview of the blur mask for a password at a level.
func BlurPreview(eng *engine.Engine, levels []levels.Level, password string, levelIndex int) string {
	if password == "" {
		return ""
	}
	level := levels[levelIndex]
	mask, err := eng.Preview(level, password)
	if err != nil {
		return ""
	}
	out := make([]rune, len(mask.Password))
	for i := range out {
		if mask.Hidden[i] {
			out[i] = 'x'
		} else {
			out[i] = '-'
		}
	}
	preview := lipgloss.NewStyle().Foreground(styles.Glow.DullFuchsia).Render(string(out))
	return "\n" + styles.DimStyle.Render("Blur preview: ") + preview
}
