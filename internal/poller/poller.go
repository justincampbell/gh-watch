package poller

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/justincampbell/gh-watch/internal/events"
	"github.com/justincampbell/gh-watch/internal/retry"
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
	ExitOn     []events.EventType

	// RetryBudget caps the total wall-clock time spent retrying transient
	// errors per poll. Zero uses the default (5 minutes).
	RetryBudget time.Duration

	// RetryNotifyOut is where retry notices are written (defaults to stderr).
	// Tests may set this to io.Discard.
	RetryNotifyOut io.Writer
}

// Run polls for state changes until the context is cancelled or a terminal event occurs.
func Run[S any](ctx context.Context, cfg Config[S]) error {
	var lastState *S

	for {
		state, err := fetchWithRetry(ctx, cfg.Fetch, cfg.RetryBudget, cfg.RetryNotifyOut)
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
			for _, exitOn := range cfg.ExitOn {
				if exitOn == events.AnyEvent || e.Event == exitOn {
					return nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(cfg.Strategy.Interval()):
		}
	}
}

const defaultRetryBudget = 5 * time.Minute

func fetchWithRetry[S any](ctx context.Context, fetch func() (*S, error), budget time.Duration, notifyOut io.Writer) (*S, error) {
	if budget <= 0 {
		budget = defaultRetryBudget
	}
	if notifyOut == nil {
		notifyOut = os.Stderr
	}

	op := func() (*S, error) {
		s, err := fetch()
		if err == nil {
			return s, nil
		}
		if !retry.IsTransient(err) {
			return nil, backoff.Permanent(err)
		}
		return nil, err
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 30 * time.Second

	notify := func(err error, next time.Duration) {
		fmt.Fprintf(notifyOut, "transient error: %v (retrying in %s)\n", err, next.Round(time.Millisecond))
	}

	return backoff.Retry(ctx, op,
		backoff.WithBackOff(bo),
		backoff.WithMaxElapsedTime(budget),
		backoff.WithNotify(notify),
	)
}
