package config

import (
	"os"

	"gopkg.in/yaml.v3"

	"github.com/alankiri/password-memorizer-tui/internal/paths"
)

type Config struct {
	Audio            bool    `yaml:"audio"`
	Welcome          Welcome `yaml:"welcome"`
	PromptedForVault bool    `yaml:"prompted_for_vault"`
}

type Welcome struct {
	StudyDays []int `yaml:"study_days"`
}

var Default = Config{
	Audio: true,
	Welcome: Welcome{
		StudyDays: []int{1, 2, 3, 4, 5, 6, 7},
	},
}

func Load() (Config, error) {
	if err := paths.EnsureDirs(); err != nil {
		return Config{}, err
	}
	p := paths.ConfigFile()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err := Save(Default); err != nil {
			return Config{}, err
		}
		return Default, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func Save(c Config) error {
	data, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}
	return os.WriteFile(paths.ConfigFile(), data, 0o644)
}
