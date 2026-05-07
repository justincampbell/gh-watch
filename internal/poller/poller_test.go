package poller

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/justincampbell/gh-watch/internal/events"
)

func TestRunRecoversFromTransientError(t *testing.T) {
	var calls int
	fetch := func() (*int, error) {
		calls++
		if calls == 1 {
			return nil, &api.HTTPError{StatusCode: 504, Message: "Gateway Timeout"}
		}
		v := 42
		return &v, nil
	}

	cfg := Config[int]{
		Fetch: fetch,
		Diff: func(_, _ *int) []events.Event {
			return []events.Event{{Event: events.InitialState}, {Event: events.CIAllPassed}}
		},
		Strategy:       &FixedStrategy{Duration: time.Hour},
		ExitOn:         []events.EventType{events.AnyEvent},
		RetryBudget:    2 * time.Second,
		RetryNotifyOut: io.Discard,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Run(ctx, cfg); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if calls < 2 {
		t.Errorf("expected fetch to be retried, got %d calls", calls)
	}
}

func TestRunDoesNotRetryPermanentError(t *testing.T) {
	var calls int
	fetch := func() (*int, error) {
		calls++
		return nil, &api.HTTPError{StatusCode: 404, Message: "Not Found"}
	}

	cfg := Config[int]{
		Fetch:          fetch,
		Diff:           func(_, _ *int) []events.Event { return nil },
		Strategy:       &FixedStrategy{Duration: time.Hour},
		RetryBudget:    2 * time.Second,
		RetryNotifyOut: io.Discard,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Run(ctx, cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var httpErr *api.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != 404 {
		t.Errorf("expected 404 HTTPError, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected exactly 1 fetch call, got %d", calls)
	}
}
