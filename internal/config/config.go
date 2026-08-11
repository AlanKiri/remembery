package config

import (
	"os"

	"gopkg.in/yaml.v3"

	"github.com/alankiri/remembery/internal/levels"
	"github.com/alankiri/remembery/internal/paths"
)

type Config struct {
	PromptedForVault bool           `yaml:"prompted_for_vault"`
	Levels           []levels.Level `yaml:"levels"`
}

var Default = Config{
	Levels: levels.Default,
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
	if len(c.Levels) == 0 {
		c.Levels = levels.Default
		if err := Save(c); err != nil {
			return Config{}, err
		}
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
