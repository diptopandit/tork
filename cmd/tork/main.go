package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/infrastructure/db"
	"github.com/diptopandit/tork/internal/infrastructure/logger"
	"github.com/diptopandit/tork/internal/infrastructure/repository"
	"github.com/diptopandit/tork/internal/infrastructure/search"
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

	// Database.
	conn, err := db.Open(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}

	// Repositories + services.
	taskRepo := repository.NewTaskRepo(conn)
	listRepo := repository.NewListRepo(conn)
	updateRepo := repository.NewUpdateRepo(conn)
	ftsSvc := search.NewFTSSearch(conn)
	taskSvc := application.NewTaskService(taskRepo, updateRepo, ftsSvc)
	listSvc := application.NewListService(listRepo)

	// Build and run the TUI.
	m := tui.NewModel(taskSvc, listSvc, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui:", err)
		os.Exit(1)
	}
}
