// Package retry classifies API errors as transient or permanent for the poller's
// retry loop.
package retry

import (
	"context"
	"errors"
	"net"

	"github.com/cli/go-gh/v2/pkg/api"
)

// IsTransient reports whether err is worth retrying — GitHub 5xx/429 responses,
// per-request timeouts, and network errors.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}

	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	return false
}
