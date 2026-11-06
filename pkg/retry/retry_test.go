package retry

import (
	"context"
	"errors"
	"net/http"
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

func TestRetryWithResultDoesNotRetryClientError(t *testing.T) {
	operationErr := &HTTPError{StatusCode: http.StatusBadRequest, Err: errors.New("bad request")}
	calls := 0

	result, err := RetryWithResult(context.Background(), func() (string, error) {
		calls++

		return "", operationErr
	}, WithRetries(3), WithInitInterval(testInterval), WithMaxInterval(testInterval))

	require.ErrorIs(t, err, operationErr)
	require.Empty(t, result)
	require.Equal(t, 1, calls)
}

func TestRetryWithResultRetriesServerError(t *testing.T) {
	operationErr := &HTTPError{StatusCode: http.StatusInternalServerError, Err: errors.New("server error")}
	calls := 0

	result, err := RetryWithResult(context.Background(), func() (string, error) {
		calls++
		if calls == 1 {
			return "", operationErr
		}

		return "done", nil
	}, WithRetries(3), WithInitInterval(testInterval), WithMaxInterval(testInterval))

	require.NoError(t, err)
	require.Equal(t, "done", result)
	require.Equal(t, 2, calls)
}

func TestRetryWithResultRetriesTooManyRequests(t *testing.T) {
	const retryAfter = 10 * time.Millisecond

	operationErr := &HTTPError{
		StatusCode: http.StatusTooManyRequests,
		RetryAfter: retryAfter,
		Err:        errors.New("rate limited"),
	}
	calls := 0
	started := time.Now()

	result, err := RetryWithResult(context.Background(), func() (string, error) {
		calls++
		if calls == 1 {
			return "", operationErr
		}

		return "done", nil
	}, WithRetries(3), WithInitInterval(testInterval), WithMaxInterval(time.Second))

	require.NoError(t, err)
	require.Equal(t, "done", result)
	require.Equal(t, 2, calls)
	require.GreaterOrEqual(t, time.Since(started), retryAfter)
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, time.November, 3, 12, 0, 0, 0, time.UTC)

	require.Equal(t, 3*time.Second, ParseRetryAfter("3", now))
	require.Equal(t, 2*time.Second, ParseRetryAfter("Tue, 03 Nov 2026 12:00:02 GMT", now))
	require.Zero(t, ParseRetryAfter("", now))
	require.Zero(t, ParseRetryAfter("0", now))
	require.Zero(t, ParseRetryAfter("-1", now))
	require.Zero(t, ParseRetryAfter("not-a-date", now))
	require.Zero(t, ParseRetryAfter("Tue, 03 Nov 2026 11:59:59 GMT", now))
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
	require.Zero(t, calls)
}

func TestRetryWithResultRetriesTimeout(t *testing.T) {
	timeoutErr := &TimeoutError{Err: context.DeadlineExceeded}
	calls := 0

	result, err := RetryWithResult(context.Background(), func() (string, error) {
		calls++
		if calls == 1 {
			return "", timeoutErr
		}

		return "done", nil
	}, WithRetries(3), WithInitInterval(testInterval), WithMaxInterval(testInterval))

	require.NoError(t, err)
	require.Equal(t, "done", result)
	require.Equal(t, 2, calls)
}

func TestRetryWithResultDoesNotRetryParentContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	time.Sleep(5 * time.Millisecond)

	calls := 0

	result, err := RetryWithResult(ctx, func() (string, error) {
		calls++

		return "", context.DeadlineExceeded
	}, WithRetries(3), WithInitInterval(testInterval), WithMaxInterval(testInterval))

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Empty(t, result)
	require.Zero(t, calls)
}
