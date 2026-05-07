package retry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

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
		{"wrapped 504", fmt.Errorf("querying PR state: %w", &api.HTTPError{StatusCode: 504}), true},
		{"wrapped 404", fmt.Errorf("querying PR state: %w", &api.HTTPError{StatusCode: 404}), false},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"wrapped deadline", fmt.Errorf("dial: %w", context.DeadlineExceeded), true},
		{"canceled", context.Canceled, false},
		{"net.OpError", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, true},
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
