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

	// Parse --remote / --local flags and apply to config.
	flagRemote, flagLocal := parseRemoteFlags(os.Args[1:])
	remoteName, _ := cfg.ResolveRemote(flagRemote, flagLocal)

	// If a specific remote was requested via flag, validate and persist.
	if remoteName != "local" && cfg.ActiveRemote(remoteName) == nil {
		fmt.Fprintf(os.Stderr, "error: remote %q not found in config (available: %v)\n", remoteName, cfg.RemoteNames())
		os.Exit(1)
	}
	if flagRemote != "" || flagLocal {
		cfg.LastRemote = remoteName
		_ = config.Save(cfg)
	}

	// Always start with local SQLite so the TUI is immediately usable.
	svc, err := bootstrap.Init(cfg, "local", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// ConnectFunc allows the TUI to switch databases on-the-fly.
	connectFunc := func(remote, password string) (*bootstrap.Services, error) {
		return bootstrap.Init(cfg, remote, password)
	}

	// Build and run the TUI. It handles remote picker + password overlay internally.
	m := tui.NewModel(svc.TaskSvc, svc.ListSvc, cfg, connectFunc, svc.DB)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		svc.DB.Close()
		fmt.Fprintln(os.Stderr, "tui:", err)
		os.Exit(1)
	}
}

// parseRemoteFlags scans args for --remote <name> or --local.
func parseRemoteFlags(args []string) (remote string, local bool) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local":
			local = true
		case "--remote":
			if i+1 < len(args) {
				remote = args[i+1]
				i++
			}
		}
	}
	return
}
