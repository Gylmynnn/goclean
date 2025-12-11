# GoClean

A terminal-based system cleaner for Arch Linux with an interactive TUI (Text User Interface).

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![Arch Linux](https://img.shields.io/badge/Arch_Linux-1793D1?style=flat&logo=arch-linux&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green.svg)

## Features

- **Interactive TUI** - Beautiful terminal interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Multiple Cleaning Categories**:
  - Package Cache - Clean old pacman package cache
  - Orphan Packages - Remove unused dependencies
  - System Cache - Clean `/var/cache`
  - User Cache - Clean `~/.cache`
  - Journal Logs - Vacuum systemd journal logs
  - Thumbnails - Remove cached thumbnails
  - Trash - Empty user trash
  - Temp Files - Clean `/tmp` and `/var/tmp`
- **Responsive Design** - Adapts to terminal size (desktop/mobile layouts)
- **Safe Operations** - Confirmation dialogs and password protection for root operations
- **Real-time Scanning** - Shows disk space usage for each category

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/Gylmynnn/goclean.git
cd goclean

# Build
go build -o goclean

# Optional: Install to PATH
sudo mv goclean /usr/local/bin/
```

### Requirements

- Go 1.25 or later
- Arch Linux (or Arch-based distributions)
- `pacman` and `paccache` (usually pre-installed on Arch)

## Usage

```bash
# Run the application
./goclean

# Show help
./goclean -h
```

## Keybindings

### Navigation
| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` | View category details |
| `Esc` / `q` | Go back / Quit |

### Selection
| Key | Action |
|-----|--------|
| `Space` | Toggle selection |
| `a` | Select all |
| `n` | Deselect all |

### Actions
| Key | Action |
|-----|--------|
| `c` | Clean selected items |
| `r` | Refresh/rescan |

### Confirm Dialog
| Key | Action |
|-----|--------|
| `←` / `→` | Switch between Cancel/Confirm |
| `y` | Confirm |
| `n` | Cancel |

## Screenshots

```
╭────────────────────────────────────────────────────────────╮
│ GoClean  Arch Linux System Cleaner                         │
╰────────────────────────────────────────────────────────────╯

  Categories

> [✓] 📦 Package Cache                                1.2 GiB
  [ ] 🔗 Orphan Packages                            256.0 MiB
  [ ] 🗄️  System Cache                              512.0 MiB
  [✓] 💾 User Cache                                   2.1 GiB
  [ ] 📋 Journal Logs                               128.0 MiB
  [ ] 🖼️  Thumbnails                                 64.0 MiB
  [ ] 🗑️  Trash                                     320.0 MiB
  [ ] 📁 Temp Files                                  48.0 MiB

╭────────────────────────────────────────────────────────────╮
│ 2 items selected  •  3.3 GiB to be freed                   │
╰────────────────────────────────────────────────────────────╯
```

<img width="1920" height="1080" alt="image" src="https://github.com/user-attachments/assets/10d1c3f0-c352-4f9d-a9ff-b192efe6f583" />

## Project Structure

```
goclean/
├── main.go                 # Entry point
├── go.mod                  # Go module definition
├── go.sum                  # Dependencies checksum
└── internal/
    ├── cleaner/
    │   ├── cleaner.go      # Cleaning operations
    │   └── types.go        # Type definitions
    ├── scanner/
    │   └── scanner.go      # System scanning
    └── tui/
        ├── tui.go          # Main TUI logic
        ├── components/
        │   └── components.go
        └── styles/
            └── styles.go
```

## How It Works

1. **Scanning** - GoClean scans your system for cleanable items across multiple categories
2. **Selection** - Navigate and select categories/items you want to clean
3. **Confirmation** - Review your selection and confirm the cleaning operation
4. **Authentication** - For system-level operations, enter your sudo password
5. **Cleaning** - GoClean removes selected items and reports the freed space

## Safety

- Operations that require root access will prompt for password
- Orphan packages are marked as "dangerous" and require explicit selection
- All cleaning operations show a confirmation dialog before proceeding
- Package cache cleaning keeps the latest version by default (`paccache -rk1`)

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Author

[Gylmynnn](https://github.com/Gylmynnn)
