package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/amyanger/livepulse/internal/tui"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	storePath := filepath.Join(home, ".livepulse", "monitors.json")
	app := tui.NewApp(storePath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.StartEngine(ctx)

	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
