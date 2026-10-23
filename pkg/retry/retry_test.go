package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testInterval = time.Millisecond

func TestRetryWithResultSuccess(t *testing.T) {
	calls := 0

	result, err := RetryWithResult(context.Background(), func() (string, error) {
		calls++

		return "done", nil
	}, WithRetries(3), WithInitInterval(testInterval), WithMaxInterval(testInterval))

	require.NoError(t, err)
	require.Equal(t, "done", result)
	require.Equal(t, 1, calls)
}

func TestRetryWithResultSuccessAfterRetries(t *testing.T) {
	operationErr := errors.New("network error")
	calls := 0

	result, err := RetryWithResult(context.Background(), func() (string, error) {
		calls++
		if calls < 3 {
			return "", operationErr
		}

		return "done", nil
	}, WithRetries(3), WithInitInterval(testInterval), WithMaxInterval(testInterval))

	require.NoError(t, err)
	require.Equal(t, "done", result)
	require.Equal(t, 3, calls)
}

func TestRetryWithResultExhausted(t *testing.T) {
	operationErr := errors.New("network error")
	calls := 0

	result, err := RetryWithResult(context.Background(), func() (string, error) {
		calls++

		return "", operationErr
	}, WithRetries(2), WithInitInterval(testInterval), WithMaxInterval(testInterval))

	require.ErrorIs(t, err, operationErr)
	require.Empty(t, result)
	require.Equal(t, 3, calls)
}

func TestRetryWithResultContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	operationErr := errors.New("network error")
	calls := 0

	result, err := RetryWithResult(ctx, func() (string, error) {
		calls++

		return "", operationErr
	}, WithRetries(3), WithInitInterval(testInterval), WithMaxInterval(testInterval))

	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, result)
	require.Equal(t, 1, calls)
}
