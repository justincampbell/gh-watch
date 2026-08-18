package poller

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestRunGivesUpWhenBudgetIsExhausted(t *testing.T) {
	fetch := func() (*int, error) {
		return nil, &api.HTTPError{StatusCode: 503, Message: "Service Unavailable"}
	}

	cfg := Config[int]{
		Fetch:          fetch,
		Diff:           func(_, _ *int) []events.Event { return nil },
		Strategy:       &FixedStrategy{Duration: time.Hour},
		RetryBudget:    time.Second,
		RetryNotifyOut: io.Discard,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var httpErr *api.HTTPError
	if err := Run(ctx, cfg); !errors.As(err, &httpErr) || httpErr.StatusCode != 503 {
		t.Errorf("expected 503 HTTPError once the budget ran out, got %v", err)
	}
}

// TestRunSurvivesOutage replays the shape of a real GitHub incident through the
// go-gh GraphQL client: a 503, a secondary rate limit, a GraphQL INTERNAL error
// returned with HTTP 200, then success.
func TestRunSurvivesOutage(t *testing.T) {
	var served int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served++
		switch served {
		case 1:
			http.Error(w, "No server is currently available to service your request.", http.StatusServiceUnavailable)
		case 2:
			w.Header().Set("Retry-After", "1")
			http.Error(w, "You have exceeded a secondary rate limit", http.StatusForbidden)
		case 3:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"errors":[{"type":"INTERNAL","message":"Something went wrong while executing your query."}]}`)
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"data":{"repository":{"pullRequest":{"number":42}}}}`)
		}
	}))
	defer srv.Close()

	client := stubGraphQLClient(t, srv.URL)
	fetch := func() (*int, error) {
		var resp struct {
			Repository struct {
				PullRequest struct{ Number int }
			}
		}
		if err := client.Do("query { repository { pullRequest { number } } }", nil, &resp); err != nil {
			return nil, err
		}
		return &resp.Repository.PullRequest.Number, nil
	}

	var stderr bytes.Buffer
	cfg := Config[int]{
		Fetch: fetch,
		Diff: func(_, _ *int) []events.Event {
			return []events.Event{{Event: events.InitialState}, {Event: events.CIAllPassed}}
		},
		Strategy:       &FixedStrategy{Duration: time.Hour},
		ExitOn:         []events.EventType{events.AnyEvent},
		RetryBudget:    30 * time.Second,
		RetryNotifyOut: &stderr,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := Run(ctx, cfg); err != nil {
		t.Fatalf("Run returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if served != 4 {
		t.Errorf("expected 4 requests (503, rate limit, GraphQL INTERNAL, success), got %d", served)
	}
	if !strings.Contains(stderr.String(), "retry after 1s") {
		t.Errorf("expected the Retry-After header to be honored, stderr:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "recovered after 3 transient error(s)") {
		t.Errorf("expected a recovery notice, stderr:\n%s", stderr.String())
	}
}

func stubGraphQLClient(t *testing.T, serverURL string) *api.GraphQLClient {
	t.Helper()

	target, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parsing stub server URL: %v", err)
	}
	client, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.localhost",
		AuthToken: "stub",
		Transport: &stubTransport{target: target},
	})
	if err != nil {
		t.Fatalf("creating stub GraphQL client: %v", err)
	}
	return client
}

// stubTransport routes GitHub API requests to a test server.
type stubTransport struct {
	target *url.URL
}

func (t *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = t.target.Scheme
	req.URL.Host = t.target.Host
	return http.DefaultTransport.RoundTrip(req)
}
