package levels

import (
	"os"

	"gopkg.in/yaml.v3"

	"github.com/alankiri/password-memorizer-tui/internal/paths"
)

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
	SessionIntervalDays        int    `yaml:"session_interval_days"`
	TypingValidationMode       string `yaml:"typing_validation_mode"`
	Description                string `yaml:"description"`
}

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

func Save(levels []Level) error {
	data, err := yaml.Marshal(&levels)
	if err != nil {
		return err
	}
	return os.WriteFile(paths.LevelsFile(), data, 0o644)
}
