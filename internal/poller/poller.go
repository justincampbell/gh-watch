package poller

import (
	"context"
	"time"

	"github.com/justincampbell/gh-watch-pr/internal/events"
	"github.com/justincampbell/gh-watch-pr/internal/pr"
)

// Strategy determines the polling interval. Pluggable for future adaptive strategies.
type Strategy interface {
	Interval() time.Duration
}

// FixedStrategy polls at a constant interval.
type FixedStrategy struct {
	Duration time.Duration
}

func (s *FixedStrategy) Interval() time.Duration {
	return s.Duration
}

// Config configures the poller.
type Config struct {
	Owner    string
	Repo     string
	Number   int
	Fetcher  pr.Fetcher
	Strategy Strategy
	OnEvents func([]events.Event)
	ExitOn   events.EventType
}

// Run polls the PR for state changes until the context is cancelled or a terminal event occurs.
func Run(ctx context.Context, cfg Config) error {
	var lastState *pr.State

	for {
		state, err := cfg.Fetcher.Fetch(cfg.Owner, cfg.Repo, cfg.Number)
		if err != nil {
			return err
		}

		detected := events.Diff(lastState, state)
		lastState = state

		if len(detected) > 0 && cfg.OnEvents != nil {
			cfg.OnEvents(detected)
		}

		// Check for terminal or exit-on events
		for _, e := range detected {
			if e.Event == events.PRMerged || e.Event == events.PRClosed {
				return nil
			}
			if cfg.ExitOn != "" && e.Event == cfg.ExitOn {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(cfg.Strategy.Interval()):
		}
	}
}
