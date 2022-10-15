package watcher

import (
	"time"

	"github.com/buglloc/f680-watcher/internal/f860"
)

type Option func(*Watcher)

func WithCheckPeriod(period time.Duration) Option {
	return func(watcher *Watcher) {
		watcher.checkPeriod = period
	}
}

func WithDHCPSources(sources map[string]f860.DHCPSourceKind) Option {
	return func(watcher *Watcher) {
		watcher.sources = sources
	}
}

func WithNotifyScript(scriptPath string) Option {
	return func(watcher *Watcher) {
		watcher.notifyScript = scriptPath
	}
}
