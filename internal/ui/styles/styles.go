package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Glow is the shared color palette used by the UI.
var Glow = struct {
	Fuchsia     lipgloss.Color
	DullFuchsia lipgloss.Color
	LightBlue   lipgloss.Color
	Blue        lipgloss.Color
	Green       lipgloss.Color
	Red         lipgloss.Color
	Yellow      lipgloss.Color
	Cream       lipgloss.Color
	Dim         lipgloss.Color
	White       lipgloss.Color
}{
	Fuchsia:     lipgloss.Color("#EE6FF8"),
	DullFuchsia: lipgloss.Color("#F793FF"),
	LightBlue:   lipgloss.Color("#8BE9FD"),
	Blue:        lipgloss.Color("#2A7FFF"),
	Green:       lipgloss.Color("#04B575"),
	Red:         lipgloss.Color("#FF4672"),
	Yellow:      lipgloss.Color("#ECFD65"),
	Cream:       lipgloss.Color("#FFFDF5"),
	Dim:         lipgloss.Color("#777777"),
	White:       lipgloss.Color("#FFFDF5"),
}

// DimStyle is the standard dim style for hints and footers.
var DimStyle = lipgloss.NewStyle().Foreground(Glow.Dim)

// Rainbow returns a string with each non-space character rendered in a
// different color.
func Rainbow(text string) string {
	colors := []string{
		"#FF0000", "#FF7F00", "#FFFF00",
		"#00FF00", "#0000FF", "#4B0082", "#9400D3",
	}
	var b strings.Builder
	i := 0
	for _, r := range text {
		if r == ' ' {
			b.WriteRune(r)
		} else {
			c := colors[i%len(colors)]
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(string(r)))
			i++
		}
	}
	return b.String()
}

// RenderTitle renders a bold title block with the given background color.
func RenderTitle(text string, bg ...lipgloss.Color) string {
	c := Glow.Fuchsia
	if len(bg) > 0 {
		c = bg[0]
	}
	return lipgloss.NewStyle().
		Bold(true).
		Background(c).
		Foreground(Glow.Cream).
		Padding(0, 1).
		Render(text)
}
