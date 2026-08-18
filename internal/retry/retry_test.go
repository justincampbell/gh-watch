package retry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestIsTransient(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"500", &api.HTTPError{StatusCode: 500}, true},
		{"502", &api.HTTPError{StatusCode: 502}, true},
		{"503", &api.HTTPError{StatusCode: 503}, true},
		{"504", &api.HTTPError{StatusCode: 504}, true},
		{"429", &api.HTTPError{StatusCode: 429}, true},
		{"401", &api.HTTPError{StatusCode: 401}, false},
		{"404", &api.HTTPError{StatusCode: 404}, false},
		{"422", &api.HTTPError{StatusCode: 422}, false},
		{"403 permission denied", &api.HTTPError{StatusCode: 403}, false},
		{"403 secondary rate limit", rateLimitedError(), true},
		{"wrapped 504", fmt.Errorf("querying PR state: %w", &api.HTTPError{StatusCode: 504}), true},
		{"wrapped 404", fmt.Errorf("querying PR state: %w", &api.HTTPError{StatusCode: 404}), false},
		{"graphql internal", graphQLError(api.GraphQLErrorItem{Type: "INTERNAL", Message: "Something went wrong while executing your query."}), true},
		{"graphql untyped incident", graphQLError(api.GraphQLErrorItem{Message: "Something went wrong while executing your query. Please include `ABC` when reporting this issue."}), true},
		{"graphql not found", graphQLError(api.GraphQLErrorItem{Type: "NOT_FOUND", Message: "Could not resolve to a Repository"}), false},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"wrapped deadline", fmt.Errorf("dial: %w", context.DeadlineExceeded), true},
		{"canceled", context.Canceled, false},
		{"net.OpError", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, true},
		{"truncated body", fmt.Errorf("reading response: %w", io.ErrUnexpectedEOF), true},
		{"empty body", json.Unmarshal([]byte(""), &struct{}{}), true},
		{"html body", json.Unmarshal([]byte("<html>502 Bad Gateway</html>"), &struct{}{}), true},
		{"schema mismatch", json.Unmarshal([]byte(`{"number":"nope"}`), &struct{ Number int }{}), false},
		{"plain error", errors.New("nope"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTransient(tc.err); got != tc.want {
				t.Errorf("IsTransient(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestRetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   time.Duration
		wantOK bool
	}{
		{"no header", &api.HTTPError{StatusCode: 429}, 0, false},
		{"seconds", httpErrorWithRetryAfter(429, "60"), 60 * time.Second, true},
		{"wrapped", fmt.Errorf("querying: %w", httpErrorWithRetryAfter(429, "5")), 5 * time.Second, true},
		{"http date is ignored", httpErrorWithRetryAfter(429, "Wed, 21 Oct 2026 07:28:00 GMT"), 0, false},
		{"not an http error", errors.New("nope"), 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := RetryAfter(tc.err)
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("RetryAfter(%v) = %v, %v; want %v, %v", tc.err, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func httpErrorWithRetryAfter(status int, value string) *api.HTTPError {
	return &api.HTTPError{StatusCode: status, Headers: http.Header{"Retry-After": []string{value}}}
}

func rateLimitedError() *api.HTTPError {
	err := httpErrorWithRetryAfter(http.StatusForbidden, "60")
	err.Message = "You have exceeded a secondary rate limit"
	return err
}

func graphQLError(items ...api.GraphQLErrorItem) *api.GraphQLError {
	return &api.GraphQLError{Errors: items}
}
