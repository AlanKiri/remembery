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

var Default = []Level{
	{
		Number:                     1,
		Color:                      "63",
		RepetitionCount:            3,
		InterAttemptDelay:          2,
		BaseBlurPercent:            0,
		BlurStepPercent:            15,
		MaxBlurPercent:             45,
		HintReductionPercent:       50,
		RequiredSessionsToProgress: 1,
		SessionIntervalDays:        1,
		TypingValidationMode:       "allow_highlight",
		Description:                "Full reveal — light blur to build comfort.",
	},
	{
		Number:                     2,
		Color:                      "121",
		RepetitionCount:            4,
		InterAttemptDelay:          3,
		BaseBlurPercent:            20,
		BlurStepPercent:            15,
		MaxBlurPercent:             55,
		HintReductionPercent:       60,
		RequiredSessionsToProgress: 2,
		SessionIntervalDays:        1,
		TypingValidationMode:       "allow_highlight",
		Description:                "Light blur, most characters still visible.",
	},
	{
		Number:                     3,
		Color:                      "226",
		RepetitionCount:            5,
		InterAttemptDelay:          4,
		BaseBlurPercent:            40,
		BlurStepPercent:            10,
		MaxBlurPercent:             65,
		HintReductionPercent:       70,
		RequiredSessionsToProgress: 2,
		SessionIntervalDays:        2,
		TypingValidationMode:       "allow_highlight",
		Description:                "Moderate blur; you need to recall more.",
	},
	{
		Number:                     4,
		Color:                      "208",
		RepetitionCount:            5,
		InterAttemptDelay:          5,
		BaseBlurPercent:            55,
		BlurStepPercent:            10,
		MaxBlurPercent:             75,
		HintReductionPercent:       80,
		RequiredSessionsToProgress: 3,
		SessionIntervalDays:        2,
		TypingValidationMode:       "allow_highlight",
		Description:                "Heavy blur; only a few characters shown.",
	},
	{
		Number:                     5,
		Color:                      "196",
		RepetitionCount:            6,
		InterAttemptDelay:          6,
		BaseBlurPercent:            70,
		BlurStepPercent:            5,
		MaxBlurPercent:             85,
		HintReductionPercent:       90,
		RequiredSessionsToProgress: 3,
		SessionIntervalDays:        3,
		TypingValidationMode:       "allow_highlight",
		Description:                "Near-full blur, but one character stays visible.",
	},
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
