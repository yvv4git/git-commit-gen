package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
)

const (
	defaultInitInterval = time.Microsecond * 100
	defaultMaxInterval  = time.Second * 3
	defaultRetries      = 3
)

type HTTPError struct {
	StatusCode int
	RetryAfter time.Duration
	Err        error
}

type TimeoutError struct {
	Err error
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	return fmt.Sprintf("HTTP status %d", e.StatusCode)
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

func (e *TimeoutError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("request timeout: %v", e.Err)
	}

	return "request timeout"
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

func ParseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0
		}

		return time.Duration(seconds) * time.Second
	}

	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0
	}

	return when.Sub(now)
}

type config struct {
	initInterval time.Duration
	maxInterval  time.Duration
	retries      uint64
}

type Option func(*config)

func WithInitInterval(value time.Duration) Option {
	return func(c *config) {
		c.initInterval = value
	}
}

func WithMaxInterval(value time.Duration) Option {
	return func(c *config) {
		c.maxInterval = value
	}
}

func WithRetries(value uint64) Option {
	return func(c *config) {
		c.retries = value
	}
}

func defaultOptions() config {
	return config{
		initInterval: defaultInitInterval,
		maxInterval:  defaultMaxInterval,
		retries:      defaultRetries,
	}
}

func RetryWithResult[T any](ctx context.Context, operation func() (T, error), opts ...Option) (T, error) {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	return retry.DoWithData(operation,
		retry.Context(ctx),
		retry.Attempts(uint(cfg.retries)+1),
		retry.Delay(cfg.initInterval),
		retry.MaxDelay(cfg.maxInterval),
		retry.DelayType(delayType),
		retry.RetryIf(isRetryable),
		retry.LastErrorOnly(true),
	)
}

func delayType(attempt uint, err error, cfg *retry.Config) time.Duration {
	if delay := retryAfter(err); delay > 0 {
		return delay
	}

	return retry.BackOffDelay(attempt, err, cfg)
}

func isRetryable(err error) bool {
	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return true
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		if httpErr.StatusCode == http.StatusTooManyRequests {
			return true
		}

		return httpErr.StatusCode < http.StatusBadRequest || httpErr.StatusCode >= http.StatusInternalServerError
	}

	return true
}

func retryAfter(err error) time.Duration {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.RetryAfter
	}

	return 0
}
