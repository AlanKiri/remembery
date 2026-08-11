package main

import (
	"strings"
	"testing"
)

func TestHelpText(t *testing.T) {
	want := []string{
		"mnemo - Terminal trainer for memorizing passwords.",
		"Usage:",
		"Available Commands:",
		"config      Edit the mnemo config file in $EDITOR",
		"Flags:",
		"-h, --help   help for mnemo",
	}
	got := helpText()
	for _, s := range want {
		if !strings.Contains(got, s) {
			t.Fatalf("helpText() missing %q\n\n%s", s, got)
		}
	}
}

func TestConfigHelpText(t *testing.T) {
	want := []string{
		"Edit the mnemo config file in $EDITOR",
		"Usage:",
		"mnemo config [flags]",
		"-h, --help   help for config",
	}
	got := configHelpText()
	for _, s := range want {
		if !strings.Contains(got, s) {
			t.Fatalf("configHelpText() missing %q\n\n%s", s, got)
		}
	}
}

func TestRunHelp(t *testing.T) {
	tests := []string{"-h", "--help", "help"}
	for _, flag := range tests {
		t.Run(flag, func(t *testing.T) {
			if err := run([]string{"mnemo", flag}); err != nil {
				t.Fatalf("run([mnemo, %q]) error: %v", flag, err)
			}
		})
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if err := run([]string{"mnemo", "foo"}); err == nil {
		t.Fatalf("expected error for unknown command, got nil")
	}
}

func TestEditConfigRequiresEditor(t *testing.T) {
	t.Setenv("EDITOR", "")
	if err := editConfig(); err == nil || !strings.Contains(err.Error(), "EDITOR") {
		t.Fatalf("expected EDITOR error, got: %v", err)
	}
}

func TestEditConfigOpensEditor(t *testing.T) {
	t.Setenv("EDITOR", "true")
	if err := editConfig(); err != nil {
		t.Fatalf("editConfig() with EDITOR=true error: %v", err)
	}
}
