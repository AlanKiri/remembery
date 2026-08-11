# Introduction

Remembery is a small TUI application created to help you memorize passwords, phone numbers, important ID's. Basically any string you can think of.
We all know how it goes most of the time. Sometimes you'd wish to remember an important password but you use it too rarely to even memorize properly. Either introduce some simple solution like a daily reminder that notifies that you should repeat it, or maybe just suffer till the end of your life.
Remembery fixes just that!

## Features

- Vim-style list navigation.
- Five configurable familiarity levels — edit `config.yaml` to adjust them to your needs.
- SQLite-backed trainers and sessions, stored in an encrypted vault.
- YAML-driven levels configuration.
- - add new list items i could forget before

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
