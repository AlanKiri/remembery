# Introduction

Remembery is a small TUI application designed to help you memorize passwords, phone numbers, important IDs — basically any string you can think of.

We all know how it goes. Sometimes you need to remember an important password, but you use it so rarely that you never really memorize it. You either adopt a simple solution, like a daily reminder, or just suffer until the end of your life.

Remembery fixes that.

## Features

- Vim-style list navigation.
- Five configurable familiarity levels — edit `config.yaml` to adjust them to your needs.
- SQLite-backed trainers and sessions, stored in an encrypted vault.
- YAML-driven level configuration.

## Installation

The easiest way to avoid any security warnings is to build from source.

### From source

If you have Go installed:

```bash
go install github.com/alankiri/remembery@latest
```

Or clone and build locally:

```bash
git clone https://github.com/alankiri/remembery.git
cd remembery
go build -o remembery .
```

### Package managers and pre-built binaries

Pre-built binaries (including those installed via Homebrew, Scoop, or downloaded from the [releases page](https://github.com/alankiri/remembery/releases)) are not code-signed. On first run, macOS and Windows may show a security warning. The binaries are safe, but the operating system cannot verify the publisher.

- **macOS:** After a first failed launch, go to **System Settings → Privacy & Security → Security** and click **Allow Anyway**. Alternatively, remove the quarantine flag:
  ```bash
  xattr -d com.apple.quarantine /path/to/remembery
  ```
- **Windows:** When SmartScreen appears, click **More info** and then **Run anyway**.

### macOS (Homebrew)

```bash
brew install alankiri/tap/remembery
```

### Windows (Scoop)

```bash
scoop bucket add remembery https://github.com/alankiri/scoop-bucket.git
scoop install remembery
```

### Linux (Snap)

```bash
snap install remembery --classic
```

### Go

If you have Go installed:

```bash
go install github.com/alankiri/remembery@latest
```

### Binaries

Download a pre-built binary from the [releases page](https://github.com/alankiri/remembery/releases).

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
