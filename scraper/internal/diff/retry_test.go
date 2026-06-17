package diff

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIsRetryableFetchError(t *testing.T) {
	cases := []struct {
		err       error
		retryable bool
	}{
		{nil, false},
		{errors.New("unexpected status code: 403"), true},
		{errors.New("unexpected status code: 429"), true},
		{errors.New("cloudflare challenge"), true},
		{errors.New("unexpected status code: 404"), false},
		{context.Canceled, false},
	}
	for _, tc := range cases {
		if got := isRetryableFetchError(tc.err); got != tc.retryable {
			t.Fatalf("isRetryableFetchError(%v) = %v, want %v", tc.err, got, tc.retryable)
		}
	}
}

func TestSleepContext_respectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepContext(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
