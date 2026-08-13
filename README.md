# Introduction

Remembery is a small TUI application designed to help you memorize passwords, phone numbers, important IDs — basically any string you can think of.

We all know how it goes. Sometimes you need to remember an important password, but you use it so rarely that you never really memorize it. You either adopt a simple solution, like a daily reminder, or just suffer until the end of your life.

Remembery fixes that.

## Features

- Vim-style list navigation.
- Five configurable familiarity levels — edit `config.yaml` to adjust them to your needs.
- SQLite-backed trainers and sessions, stored in an encrypted vault.
- YAML-driven level configuration.

## Data and configuration

Paths are resolved automatically:

- macOS / Linux: `~/.config/remembery/`
- Windows: `%APPDATA%\remembery\`

Files created on first run:

- `config.yaml` — app settings and level definitions.
- `data.db` — SQLite database for trainers and sessions.

## Roadmap

- OS-level notifications
- Custom cross-platform audio backend
- Tags, colors, import/export
