package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/NasaVasa/botty/internal/app"
	"github.com/NasaVasa/botty/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config:", err)
		os.Exit(1)
	}

	application, err := app.New(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to initialize app:", err)
		os.Exit(1)
	}
	defer application.Shutdown()

	if err := application.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "application error:", err)
		os.Exit(1)
	}
}
