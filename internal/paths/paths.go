package paths

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/alankiri/password-memorizer-tui/internal/consts"
)

func home() string {
	h := os.Getenv("HOME")
	if h == "" {
		h = os.Getenv("USERPROFILE")
	}
	return h
}

func configRoot() string {
	if runtime.GOOS == "windows" {
		root := os.Getenv("APPDATA")
		if root == "" {
			root = home()
		}
		return root
	}
	root := os.Getenv("XDG_CONFIG_HOME")
	if root == "" {
		root = filepath.Join(home(), ".config")
	}
	return root
}

func ConfigDir() string {
	return filepath.Join(configRoot(), consts.AppName)
}

func DataDir() string {
	return ConfigDir()
}

func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func LevelsFile() string {
	return filepath.Join(ConfigDir(), "levels.yaml")
}

func DBFile() string {
	return filepath.Join(DataDir(), "data.db")
}

func EnsureDirs() error {
	for _, d := range []string{ConfigDir(), DataDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}
