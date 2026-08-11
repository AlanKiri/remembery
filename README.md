# passmem

A cross-platform, config-driven terminal trainer in Go that teaches user-supplied passwords through progressive blur, timed repetitions, and vim-style navigation.

## Features

- Vim-style list navigation (j/k, g/G).
- Five configurable familiarity levels with colored blur.
- Real-time `allow_highlight` typing mode.
- Timed inter-attempt delay with a terminal beep.
- SQLite-backed trainers and sessions.
- YAML-driven app configuration and level definitions.
- Level-up offers when a trainer completes the required sessions.

## Tech stack

- Go 1.22+
- `charmbracelet/bubbletea`, `charmbracelet/lipgloss`, `charmbracelet/bubbles`
- `modernc.org/sqlite` (pure Go, no cgo)
- `gopkg.in/yaml.v3`

## Install

Requires a Go toolchain.

```bash
go build .
```

This produces a `password-memorizer-tui` binary. You can also `go install` or copy the binary anywhere.

## Run

```bash
./password-memorizer-tui
```

Or with `go run`:

```bash
go run .
```

The first run creates the configuration and data directories and writes a default `config.yaml` file.

## Data and configuration

Paths are resolved automatically:

- macOS / Linux: `~/.config/passmem/`
- Windows: `%APPDATA%\passmem\`

Files created on first run:

- `config.yaml` — app settings and level definitions.
- `data.db` — SQLite database for trainers and sessions.

## Key bindings

### Welcome

| Key   | Action               |
| ----- | -------------------- |
| Enter | Continue to the list |
| q     | Quit                 |

### Main list

| Key      | Action                  |
| -------- | ----------------------- |
| j / down | Move down               |
| k / up   | Move up                 |
| g        | Go to top               |
| G        | Go to bottom            |
| n        | New trainer             |
| d        | Delete selected trainer |
| r        | Refresh list            |
| Enter    | Start training          |
| q        | Quit                    |

### New trainer

| Key                | Action                                      |
| ------------------ | ------------------------------------------- |
| Tab                | Switch focus (label, password, level)       |
| up / down or j / k | Change level when the level list is focused |
| Enter              | Save trainer                                |
| Esc                | Cancel                                      |

### Training

| Key       | Action                                           |
| --------- | ------------------------------------------------ |
| type      | Type the password                                |
| Enter     | Submit the current attempt                       |
| Backspace | Remove the last character                        |
| Ctrl+H    | Show the full password as a hint and clear input |
| Esc       | Quit the session without recording               |

### Delete / Level-up prompts

| Key     | Action  |
| ------- | ------- |
| y       | Confirm |
| n / Esc | Cancel  |

## Configuration files

### config.yaml

```yaml
audio: true
welcome:
  study_days:
    - 1
    - 2
    - 3
    - 4
    - 5
    - 6
    - 7
```

- `audio` — whether the terminal beep (`\a`) is used.
- `welcome.study_days` — list of weekdays used for welcome stats.

## Blur formula

```text
effective_blur = clamp(base_blur + sessions_at_level * blur_step, 0, max_blur)
hidden_count   = min(n - 1, ceil(n * effective_blur / 100))
```

Random positions are hidden for each repetition. The `n - 1` rule guarantees at least one visible character at every level.

## Project structure

```text
internal/
  beep/     Terminal beep interface
  config/   App config loading
  consts/   Exported app name
  engine/   Blur/hint math and level-up logic
  levels/   Level defaults and types
  paths/    Config / data path helpers
  store/    SQLite repository
  tui/      Bubble Tea UI
main.go
```

## Building

```bash
go build .
```

Cross-platform builds are supported because the app uses only pure-Go dependencies:

```bash
GOOS=linux GOARCH=amd64 go build .
GOOS=windows GOARCH=amd64 go build .
```

## Roadmap / out of MVP

- Master-key / encrypted store
- OS-level notifications
- Custom cross-platform audio backend
- Tags, colors, import/export
- Approximate blur progression preview before level-up
