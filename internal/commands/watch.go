package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/buglloc/f680-watcher/internal/config"
	"github.com/buglloc/f680-watcher/internal/f860"
	"github.com/buglloc/f680-watcher/internal/watcher"
)

var watchCmd = &cobra.Command{
	Use:           "watch",
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         "Starts watcher",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := config.LoadConfig(rootArgs.cfgPath)
		if err != nil {
			return fmt.Errorf("unable to parse config: %w", err)
		}

		if cfg.Debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}

		f860c, err := f860.NewClient(
			f860.RouterConfig{
				Upstream: cfg.Router.Upstream,
				Username: cfg.Router.Username,
				Password: cfg.Router.Password,
			},
			f860.WithDebug(cfg.Debug),
			f860.WithTimeout(cfg.Router.Timeout),
		)
		if err != nil {
			return fmt.Errorf("unable to create f860 client: %w", err)
		}

		instance := watcher.NewWatcher(
			f860c,
			watcher.WithCheckPeriod(cfg.CheckPeriod),
			watcher.WithDHCPSources(cfg.DHCPSources),
			watcher.WithNotifyScript(cfg.NotifyScript),
		)

		errChan := make(chan error, 1)
		okChan := make(chan struct{}, 1)
		go func() {
			err := instance.Watch()
			if err != nil {
				errChan <- err
				log.Error().Err(err).Msg("start failed")
			} else {
				okChan <- struct{}{}
			}
		}()

		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-stopChan:
			log.Info().Msg("shutting down gracefully by signal")

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			instance.Shutdown(ctx)
		case <-okChan:
		case <-errChan:
		}

		return nil
	},
}
