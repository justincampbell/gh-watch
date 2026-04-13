package poller

import (
	"context"
	"time"

	"github.com/justincampbell/gh-watch/internal/events"
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

// Config configures the poller for any watchable resource type S.
type Config[S any] struct {
	Fetch      func() (*S, error)
	Diff       func(old, new *S) []events.Event
	IsTerminal func(events.Event) bool
	Strategy   Strategy
	OnEvents   func([]events.Event)
	ExitOn     events.EventType
}

// Run polls for state changes until the context is cancelled or a terminal event occurs.
func Run[S any](ctx context.Context, cfg Config[S]) error {
	var lastState *S

	for {
		state, err := cfg.Fetch()
		if err != nil {
			return err
		}

		detected := cfg.Diff(lastState, state)
		lastState = state

		if len(detected) > 0 && cfg.OnEvents != nil {
			cfg.OnEvents(detected)
		}

		for _, e := range detected {
			if e.Event == events.InitialState {
				continue
			}
			if cfg.IsTerminal != nil && cfg.IsTerminal(e) {
				return nil
			}
			if cfg.ExitOn != "" && (cfg.ExitOn == events.AnyEvent || e.Event == cfg.ExitOn) {
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
