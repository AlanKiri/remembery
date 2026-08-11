package levels

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
	RequiredSessionsToProgress int    `yaml:"required_sessions_to_progress"`
	SessionIntervalHours       int    `yaml:"session_interval_hours"`
	Description                string `yaml:"description"`
}
