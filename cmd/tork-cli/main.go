package main

import (
	"fmt"
	"os"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	"github.com/diptopandit/tork/internal/infrastructure/db"
	"github.com/diptopandit/tork/internal/infrastructure/repository"
	"github.com/diptopandit/tork/internal/infrastructure/search"
	cli "github.com/diptopandit/tork/internal/interface/cli"
)

func main() {
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

	taskRepo := repository.NewTaskRepo(conn)
	listRepo := repository.NewListRepo(conn)
	updateRepo := repository.NewUpdateRepo(conn)
	ftsSvc := search.NewFTSSearch(conn)
	taskSvc := application.NewTaskService(taskRepo, updateRepo, ftsSvc)
	listSvc := application.NewListService(listRepo)

	root := cli.NewRootCmd(taskSvc, listSvc)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
