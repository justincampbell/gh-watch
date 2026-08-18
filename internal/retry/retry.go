// Package retry classifies API errors as transient or permanent for the poller's
// retry loop.
package retry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// IsTransient reports whether err is worth retrying — GitHub 5xx/429 responses,
// secondary rate limits, GraphQL internal errors, per-request timeouts, network
// errors, and truncated or malformed bodies.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}

	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		switch {
		case httpErr.StatusCode >= 500, httpErr.StatusCode == http.StatusTooManyRequests:
			return true
		case httpErr.StatusCode == http.StatusForbidden:
			// GitHub reports secondary rate limits as 403; a Retry-After header
			// is what distinguishes one from a real permission error.
			_, ok := RetryAfter(err)
			return ok
		}
		return false
	}

	// GraphQL answers with HTTP 200 even when the query failed server-side.
	var graphQLErr *api.GraphQLError
	if errors.As(err, &graphQLErr) {
		return isGraphQLInternal(graphQLErr)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Truncated or non-JSON body, e.g. a gateway answering in GitHub's place.
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	var syntaxErr *json.SyntaxError
	return errors.As(err, &syntaxErr)
}

// RetryAfter returns how long GitHub asked us to wait, if it said.
func RetryAfter(err error) (time.Duration, bool) {
	var httpErr *api.HTTPError
	if !errors.As(err, &httpErr) {
		return 0, false
	}

	seconds, convErr := strconv.Atoi(strings.TrimSpace(httpErr.Headers.Get("Retry-After")))
	if convErr != nil || seconds < 0 {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}

func isGraphQLInternal(err *api.GraphQLError) bool {
	for _, item := range err.Errors {
		if item.Type == "INTERNAL" || item.Type == "SERVICE_UNAVAILABLE" {
			return true
		}
		if strings.HasPrefix(item.Message, "Something went wrong while executing your query") {
			return true
		}
	}
	return false
}
