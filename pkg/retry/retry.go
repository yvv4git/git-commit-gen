package retry

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

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
		initInterval: time.Microsecond * 100,
		maxInterval:  time.Second * 3,
		retries:      3,
	}
}

func RetryWithResult[T any](ctx context.Context, operation func() (T, error), opts ...Option) (T, error) {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	bk := backoff.NewExponentialBackOff()
	bk.InitialInterval = cfg.initInterval
	bk.MaxInterval = cfg.maxInterval

	var (
		zero   T
		result T
	)

	backoffWithContext := backoff.WithContext(bk, ctx)
	backoffWithMaxRetries := backoff.WithMaxRetries(backoffWithContext, cfg.retries)

	err := backoff.Retry(func() error {
		var err error
		result, err = operation()
		return err
	}, backoffWithMaxRetries)

	if err != nil {
		return zero, err
	}

	return result, nil
}
