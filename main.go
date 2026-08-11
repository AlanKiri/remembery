package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/alankiri/remembery/internal/config"
	"github.com/alankiri/remembery/internal/consts"
	"github.com/alankiri/remembery/internal/paths"
	"github.com/alankiri/remembery/internal/ui"
)

func main() {
	if err := run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) == 1 {
		return ui.Run()
	}
	switch args[1] {
	case "-h", "--help", "help":
		fmt.Print(helpText())
		return nil
	case "config":
		if hasHelp(args[2:]) {
			fmt.Print(configHelpText())
			return nil
		}
		if len(args) > 2 {
			return fmt.Errorf("unknown argument %q for command %q", args[2], "config")
		}
		return editConfig()
	default:
		return fmt.Errorf("unknown command %q", args[1])
	}
}

func hasHelp(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

func editConfig() error {
	if _, err := config.Load(); err != nil {
		return err
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return fmt.Errorf("EDITOR environment variable is not set")
	}
	cmd := exec.Command(editor, paths.ConfigFile())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func helpText() string {
	return fmt.Sprintf(`%s - Terminal trainer for memorizing passwords.

Usage:
  %s [command]
  %s [flags]

Available Commands:
  config      Edit the %s config file in $EDITOR
  help        Help about any command

Flags:
  -h, --help   help for %s

Use "%s [command] --help" for more information about a command.
`, consts.AppName, consts.AppName, consts.AppName, consts.AppName, consts.AppName, consts.AppName)
}

func configHelpText() string {
	return fmt.Sprintf(`Edit the %s config file in $EDITOR.

Usage:
  %s config [flags]

Flags:
  -h, --help   help for config
`, consts.AppName, consts.AppName)
}
