package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/diptopandit/tork/internal/infrastructure/bootstrap"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/infrastructure/logger"
	"github.com/diptopandit/tork/internal/interface/tui"
)

func main() {
	// Config (loaded first so data_dir is available).
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	dataDir, err := cfg.ResolveDataDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "data dir:", err)
		os.Exit(1)
	}

	// Logger (best-effort; failures are non-fatal).
	log, err := logger.New(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warn: logger init:", err)
	}
	if log != nil {
		defer log.Sync()
	}

	// Database + services (SQLite or MySQL based on config).
	svc, err := bootstrap.Init(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer svc.DB.Close()

	// Build and run the TUI.
	m := tui.NewModel(svc.TaskSvc, svc.ListSvc, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui:", err)
		os.Exit(1)
	}
}
