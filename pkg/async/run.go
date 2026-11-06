package async

import (
	"context"
	"errors"
	"time"
)

var ErrTimeout = errors.New("operation timed out")

func RunWithTimeoutAndResult[T any](ctx context.Context, fn func(context.Context) (T, error), timeout time.Duration) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	chRes := make(chan T, 1)
	chErr := make(chan error, 1)

	go func() {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := fn(ctx)
			if err != nil {
				chErr <- err
				chRes <- data
				return
			}
			chRes <- data
		}
	}()

	select {
	case <-ctx.Done():
		var zero T
		if errors.Is(ctx.Err(), context.Canceled) {
			return zero, context.Canceled
		}
		return zero, ErrTimeout
	case err := <-chErr:
		res := <-chRes
		return res, err
	case res := <-chRes:
		return res, nil
	}
}

func RunWithTimeout(ctx context.Context, fn func(context.Context) error, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	chErr := make(chan error, 1)

	go func() {
		select {
		case <-ctx.Done():
			return
		default:
			err := fn(ctx)
			if err != nil {
				chErr <- err
			}
			close(chErr)
		}
	}()

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return context.Canceled
		}
		return ErrTimeout
	case err := <-chErr:
		return err
	}
}
