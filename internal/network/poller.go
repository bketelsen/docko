package network

import (
	"context"
	"log/slog"
	"time"
)

// Poller runs periodic sync for continuous-sync sources.
type Poller struct {
	service  *Service
	interval time.Duration
}

// NewPoller creates a new poller.
func NewPoller(service *Service, interval time.Duration) *Poller {
	return &Poller{
		service:  service,
		interval: interval,
	}
}

// Run starts the polling loop. Blocks until context is cancelled.
func (p *Poller) Run(ctx context.Context) error {
	// Run initial sync
	p.syncContinuousSources(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.syncContinuousSources(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// syncContinuousSources syncs all sources with continuous_sync enabled.
func (p *Poller) syncContinuousSources(ctx context.Context) {
	sources, err := p.service.db.Queries.ListContinuousSyncSources(ctx)
	if err != nil {
		slog.Error("failed to list continuous sync sources", "error", err)
		return
	}

	if len(sources) == 0 {
		return
	}

	slog.Debug("running scheduled sync", "sources", len(sources))

	for _, source := range sources {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if _, err := p.service.SyncSource(ctx, source.ID); err != nil {
			slog.Warn("scheduled sync failed",
				"source", source.Name,
				"error", err,
			)
			// Continue with other sources
		}
	}
}

// TriggerSync manually triggers a sync for all continuous sources.
// Can be called from handler for "Sync All" button.
func (p *Poller) TriggerSync(ctx context.Context) {
	p.syncContinuousSources(ctx)
}
