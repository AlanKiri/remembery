package levels

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/alankiri/password-memorizer-tui/internal/paths"
)

// Level holds the parameters for one difficulty level in the memorizer.
// These values drive the training mask, timing, and scheduling behavior.
type Level struct {
	Number                     int    `yaml:"number"`
	Color                      string `yaml:"color"`
	RepetitionCount            int    `yaml:"repetition_count"`
	InterAttemptDelay          int    `yaml:"inter_attempt_delay"`
	BaseBlurPercent            int    `yaml:"base_blur_percent"`
	BlurStepPercent            int    `yaml:"blur_step_percent"`
	MaxBlurPercent             int    `yaml:"max_blur_percent"`
	HintReductionPercent       int    `yaml:"hint_reduction_percent"`
	RequiredSessionsToProgress int    `yaml:"required_sessions_to_progress"`
	SessionIntervalHours       int    `yaml:"session_interval_hours"`
	TypingValidationMode       string `yaml:"typing_validation_mode"`
	Description                string `yaml:"description"`
}

// Load reads the user levels yaml, creating defaults if it is missing.
func Load() ([]Level, error) {
	if err := paths.EnsureDirs(); err != nil {
		return nil, err
	}
	p := paths.LevelsFile()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err := Save(Default); err != nil {
			return nil, err
		}
		return Default, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var levels []Level
	if err := yaml.Unmarshal(data, &levels); err != nil {
		return nil, err
	}
	return levels, nil
}

// FieldDocs pairs every Level field with a one-line explanation. These are
// written as comments at the top of levels.yaml so users can edit by hand.
var FieldDocs = []struct {
	Name string
	Desc string
}{
	{Name: "number", Desc: "Level identifier, must be unique and increasing."},
	{Name: "color", Desc: "Hex color used for the level in the UI."},
	{Name: "repetition_count", Desc: "How many times the password is typed in a complete training session."},
	{Name: "inter_attempt_delay", Desc: "Milliseconds to wait before the next attempt is allowed."},
	{Name: "base_blur_percent", Desc: "Percentage of characters blurred at the start of a level (0-100)."},
	{Name: "blur_step_percent", Desc: "How much blur increases after each failed attempt (0-100)."},
	{Name: "max_blur_percent", Desc: "Maximum blur that can be applied to a password (0-100)."},
	{Name: "hint_reduction_percent", Desc: "How much the hint shrinks after each attempt (0-100)."},
	{Name: "required_sessions_to_progress", Desc: "How many counted training sessions are needed to advance."},
	{Name: "session_interval_hours", Desc: "Minimum hours between counted training sessions."},
	{Name: "typing_validation_mode", Desc: "Training strictness. 'allow_highlight' is the current default."},
	{Name: "description", Desc: "Short description shown in the settings screen."},
}

// Save writes levels to the yaml file, adding a header comment that
// documents every field so the file is understandable without the app.
func Save(levels []Level) error {
	var b strings.Builder
	b.WriteString("# passmem level configuration\n")
	b.WriteString("#\n")
	for _, d := range FieldDocs {
		b.WriteString(fmt.Sprintf("# %s: %s\n", d.Name, d.Desc))
	}
	b.WriteString("#\n")
	data, err := yaml.Marshal(&levels)
	if err != nil {
		return err
	}
	b.Write(data)
	return os.WriteFile(paths.LevelsFile(), []byte(b.String()), 0o644)
}
