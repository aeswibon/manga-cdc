package diff

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

func isRetryableFetchError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	msg := strings.ToLower(err.Error())
	retryMarkers := []string{
		"unexpected status code: 403",
		"unexpected status code: 429",
		"unexpected status code: 502",
		"unexpected status code: 503",
		"unexpected status code: 520",
		"unexpected status code: 521",
		"unexpected status code: 522",
		"unexpected status code: 523",
		"unexpected status code: 524",
		"cloudflare",
		"connection reset",
		"connection refused",
		"i/o timeout",
		"tls handshake timeout",
		"eof",
		"temporary failure",
	}
	for _, marker := range retryMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
