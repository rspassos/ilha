package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rspassos/ilha/services/player-stats-api/internal/bootstrap"
	"github.com/rspassos/ilha/services/player-stats-api/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "player-stats-api failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var bootstrapOnly bool
	var envFilePath string
	flag.BoolVar(&bootstrapOnly, "bootstrap-only", false, "run startup validation and exit")
	flag.StringVar(&envFilePath, "env-file", config.DefaultEnvFile, "path to the optional env file")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.NewApp(ctx, bootstrap.Options{
		BootstrapOnly: bootstrapOnly,
		EnvFilePath:   envFilePath,
	})
	if err != nil {
		return err
	}

	return app.Run(ctx)
}
