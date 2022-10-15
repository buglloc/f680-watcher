package watcher

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/buglloc/f680-watcher/internal/f860"
)

const (
	DefaultCheckPeriod = 1 * time.Minute
)

type Watcher struct {
	f860c        *f860.Client
	checkPeriod  time.Duration
	sources      map[string]f860.DHCPSourceKind
	notifyScript string
	ctx          context.Context
	cancelCtx    context.CancelFunc
	closed       chan struct{}
}

func NewWatcher(f860c *f860.Client, opts ...Option) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())

	out := &Watcher{
		f860c:       f860c,
		checkPeriod: DefaultCheckPeriod,
		ctx:         ctx,
		cancelCtx:   cancel,
		closed:      make(chan struct{}),
	}

	for _, opt := range opts {
		opt(out)
	}

	return out
}

func (w *Watcher) Watch() error {
	defer close(w.closed)

	if err := w.Sync(w.ctx); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}

	ticker := time.NewTicker(w.checkPeriod)
	for {
		select {
		case <-w.ctx.Done():
			ticker.Stop()
			return nil
		case <-ticker.C:
			log.Info().Msg("starts syncing")
			if err := w.Sync(w.ctx); err != nil {
				log.Error().Err(err).Msg("sync failed")
			}
		}
	}
}

func (w *Watcher) Sync(ctx context.Context) error {
	if err := w.f860c.Reset(); err != nil {
		return fmt.Errorf("f860c.Reset: %w", err)
	}

	log.Info().Msg("try to login into router")
	authorized, err := w.f860c.Login(ctx)
	if err != nil {
		return fmt.Errorf("f860c.Login: %w", err)
	}

	if !authorized {
		return errors.New("unable to authorize")
	}

	log.Info().Msg("load router DHCP sources")
	routerSources, err := w.f860c.LanDevDHCPSources(ctx)
	if err != nil {
		return fmt.Errorf("f860c.LanDevDHCPSources: %w", err)
	}

	toUpdate := w.diffDHCPSources(routerSources)
	if len(toUpdate) == 0 {
		log.Info().Msg("nothing to update")
		return nil
	}

	log.Info().Msg("update sources")
	for _, source := range toUpdate {
		sourceName := source.VendorClassID
		logger := log.With().Str("source", sourceName).Logger()

		if err := w.f860c.UpdateLanDevDHCPSource(ctx, toUpdate...); err != nil {
			logger.Error().Err(err).Msg("unable to update source")
			continue
		}

		logger.Info().Msg("updated")

		if w.notifyScript == "" {
			continue
		}

		logger.Info().
			Str("notify_script", w.notifyScript).
			Msg("calling notify script")
		cmd := exec.CommandContext(ctx, w.notifyScript, sourceName)
		if err := cmd.Run(); err != nil {
			logger.Error().Err(err).Msg("notify failed")
		}
		logger.Info().Msg("notified")
	}

	return nil
}

func (w *Watcher) diffDHCPSources(currentSources []f860.DevDHCPSource) []f860.DevDHCPSource {
	var out []f860.DevDHCPSource
	for _, source := range currentSources {
		expected, exists := w.sources[source.VendorClassID]
		if !exists {
			log.Info().
				Str("source", source.VendorClassID).
				Msg("skip unwatched source")
			continue
		}

		if expected == source.ProcFlag {
			continue
		}

		log.Info().
			Str("source_id", source.ID).
			Str("source", source.VendorClassID).
			Str("expected", expected.String()).
			Str("actual", source.ProcFlag.String()).
			Msg("schedule source update")

		source.ProcFlag = expected
		out = append(out, source)
	}

	return out
}

func (w *Watcher) Shutdown(ctx context.Context) {
	w.cancelCtx()

	select {
	case <-ctx.Done():
	case <-w.closed:
	}
}
