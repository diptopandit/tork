package main

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/diptopandit/tork/internal/infrastructure/bootstrap"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/infrastructure/logger"
	cli "github.com/diptopandit/tork/internal/interface/cli"
)

// Set via -ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	// Parse --remote / --local flags before Cobra gets them.
	flagRemote, flagLocal := parseRemoteFlags(os.Args[1:])

	remoteName, needsPicker := cfg.ResolveRemote(flagRemote, flagLocal)
	if needsPicker {
		// CLI can't show a picker — default to local with a hint.
		fmt.Fprintln(os.Stderr, "hint: use --remote <name> to connect to a remote database")
		fmt.Fprintf(os.Stderr, "      configured remotes: %v\n", cfg.RemoteNames())
		remoteName = "local"
	}

	// Validate the named remote exists.
	if remoteName != "local" && cfg.ActiveRemote(remoteName) == nil {
		fmt.Fprintf(os.Stderr, "error: remote %q not found in config (available: %v)\n", remoteName, cfg.RemoteNames())
		os.Exit(1)
	}

	// Prompt for password if using a remote.
	var password string
	if rc := cfg.ActiveRemote(remoteName); rc != nil {
		fmt.Fprintf(os.Stderr, "Password for %s: ", rc.Label())
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading password:", err)
			os.Exit(1)
		}
		password = string(raw)
	}

	// Persist choice.
	if cfg.LastRemote != remoteName {
		cfg.LastRemote = remoteName
		_ = config.Save(cfg)
	}

	// Logger (best-effort; failures are non-fatal).
	dataDir, _ := cfg.ResolveDataDir()
	log, logErr := logger.New(dataDir, logger.ParseLevel(cfg.LogLevel))
	if logErr != nil {
		fmt.Fprintln(os.Stderr, "warn: logger init:", logErr)
	}
	if log != nil {
		defer log.Sync()
	}

	svc, err := bootstrap.Init(cfg, remoteName, password, log)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer svc.DB.Close()

	root := cli.NewRootCmd(svc.TaskSvc, svc.ListSvc, cfg, fmt.Sprintf("%s (%s, %s)", version, commit, date))
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// parseRemoteFlags scans args for --remote <name> or --local before Cobra runs.
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
