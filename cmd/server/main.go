// Command hyperfocus is the entrypoint for the DBD live stream history
// tracker. It reads ./config.yaml, runs database migrations automatically, then
// starts the HTTP server, the poller, the VOD resolver and the retention loop.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"hyperfocus/internal/config"
	"hyperfocus/internal/container"
	"hyperfocus/internal/pkg/logger"
)

// configPath is the fixed configuration file the application always reads.
const configPath = "config.yaml"

func main() {
	root := &cobra.Command{
		Use:   "hyperfocus",
		Short: "Dead by Daylight live-streamer history tracker",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			log := logger.New(cfg.Log.Level, cfg.Log.Format)

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			app, err := container.Build(ctx, cfg, log)
			if err != nil {
				return err
			}

			log.Info("hyperfocus starting",
				slog.String("version", container.Version),
				slog.String("addr", cfg.Service.HTTPAddr))

			errCh := make(chan error, 1)
			go func() { errCh <- app.Server.ListenAndServe() }()

			select {
			case <-ctx.Done():
				log.Info("shutdown signal received")
			case err := <-errCh:
				if err != nil {
					return err
				}
			}

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := app.Server.Shutdown(shutdownCtx); err != nil {
				log.Warn("graceful shutdown failed", slog.Any("error", err))
			}
			app.Shutdown()
			log.Info("hyperfocus stopped")
			return nil
		},
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
