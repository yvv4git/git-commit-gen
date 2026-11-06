package async

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunWithTimeoutSuccess(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	err := RunWithTimeout(ctx, fn, 100*time.Millisecond)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRunWithTimeoutAndResultSuccess(t *testing.T) {
	ctx := context.Background()
	expectedRes := "test result"
	fn := func(ctx context.Context) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return expectedRes, nil
	}

	res, err := RunWithTimeoutAndResult(ctx, fn, 100*time.Millisecond)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if res != expectedRes {
		t.Errorf("Expected result %v, got %v", expectedRes, res)
	}
}

func TestRunWithTimeoutAndResultError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	fn := func(ctx context.Context) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "", expectedErr
	}

	res, err := RunWithTimeoutAndResult(ctx, fn, 100*time.Millisecond)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if res != "" {
		t.Errorf("Expected empty result, got %v", res)
	}
}

func TestRunWithTimeoutAndResultTimeout(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "result", nil
	}

	res, err := RunWithTimeoutAndResult(ctx, fn, 10*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected ErrTimeout, got %v", err)
	}
	if res != "" {
		t.Errorf("Expected empty result, got %v", res)
	}
}

func TestRunWithTimeoutAndResultContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fn := func(ctx context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "result", nil
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	res, err := RunWithTimeoutAndResult(ctx, fn, 50*time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	if res != "" {
		t.Errorf("Expected empty result, got %v", res)
	}
}

func TestRunWithTimeoutAndResultEdgeCases(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) (string, error) {
		return "result", nil
	}

	// Test zero timeout
	res, err := RunWithTimeoutAndResult(ctx, fn, 0)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected ErrTimeout with zero timeout, got %v", err)
	}
	if res != "" {
		t.Errorf("Expected empty result with zero timeout, got %v", res)
	}

	// Test negative timeout
	res, err = RunWithTimeoutAndResult(ctx, fn, -1*time.Second)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected ErrTimeout with negative timeout, got %v", err)
	}
	if res != "" {
		t.Errorf("Expected empty result with negative timeout, got %v", res)
	}
}

func TestRunWithTimeoutError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	fn := func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return expectedErr
	}

	err := RunWithTimeout(ctx, fn, 100*time.Millisecond)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRunWithTimeoutActualTimeout(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	err := RunWithTimeout(ctx, fn, 10*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected ErrTimeout, got %v", err)
	}
}

func TestRunWithTimeoutZeroTimeout(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}

	err := RunWithTimeout(ctx, fn, 0)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected ErrTimeout with zero timeout, got %v", err)
	}
}

func TestRunWithTimeoutNegativeTimeout(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}

	err := RunWithTimeout(ctx, fn, -1*time.Second)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected ErrTimeout with negative timeout, got %v", err)
	}
}

func TestRunWithTimeoutContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fn := func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	// Cancel the context after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := RunWithTimeout(ctx, fn, 50*time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}
