package common

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/password-memorizer-tui/internal/engine"
	"github.com/alankiri/password-memorizer-tui/internal/levels"
	"github.com/alankiri/password-memorizer-tui/internal/ui/styles"
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

// WrapWords wraps text so that no line exceeds the given width.
func WrapWords(text string, width int) string {
	words := strings.Fields(text)
	var b strings.Builder
	var line string
	for _, w := range words {
		if line != "" && len(line)+1+len(w) > width {
			b.WriteString(line)
			b.WriteString("\n")
			line = w
		} else {
			if line != "" {
				line += " "
			}
			line += w
		}
	}
	if line != "" {
		b.WriteString(line)
	}
	return b.String()
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
	preview := lipgloss.NewStyle().Foreground(styles.Glow.DullFuchsia).Render(mask.Blurred)
	return "\n" + styles.DimStyle.Render("Blur preview: ") + preview
}
