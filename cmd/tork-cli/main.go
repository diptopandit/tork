package main

import (
	"fmt"
	"os"

	"github.com/diptopandit/tork/internal/infrastructure/bootstrap"
	"github.com/diptopandit/tork/internal/infrastructure/config"
	cli "github.com/diptopandit/tork/internal/interface/cli"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	svc, err := bootstrap.Init(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer svc.DB.Close()

	root := cli.NewRootCmd(svc.TaskSvc, svc.ListSvc, cfg)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
