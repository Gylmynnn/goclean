package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Gylmynnn/goclean/internal/tui"
)

const helpText = `GoClean - Arch Linux System Cleaner

Usage: goclean [options]

Options:
  -h, --help    Show this help message

Keybindings:
  Navigation:
    ↑/k         Move cursor up
    ↓/j         Move cursor down
    enter       View category details
    esc/q       Go back / Quit

  Selection:
    space       Toggle selection
    a           Select all
    n           Deselect all

  Actions:
    c           Clean selected items
    r           Refresh/rescan

  Confirm Dialog:
    ←/→         Switch between Cancel/Confirm
    y           Confirm
    n           Cancel
`

func main() {
	help := flag.Bool("h", false, "Show help")
	flag.BoolVar(help, "help", false, "Show help")
	flag.Parse()

	if *help {
		fmt.Print(helpText)
		os.Exit(0)
	}

	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
