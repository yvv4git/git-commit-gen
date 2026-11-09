package checker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvv4git/git-commit-gen/internal/ports"
)

var errCheck = errors.New("check failed")

type fakeChecker struct {
	result *ports.ProviderCheckResult
	err    error
	calls  int
}

func (f *fakeChecker) CheckProvider(_ context.Context) (*ports.ProviderCheckResult, error) {
	f.calls++

	return f.result, f.err
}

func TestCheckSuccess(t *testing.T) {
	port := &fakeChecker{result: &ports.ProviderCheckResult{Value: "I am a model"}}
	checker := NewChecker(port)

	result, err := checker.Check(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "I am a model", result.Value)
	assert.Equal(t, 1, port.calls)
}

func TestCheckError(t *testing.T) {
	port := &fakeChecker{err: errCheck}
	checker := NewChecker(port)

	_, err := checker.Check(context.Background())

	require.ErrorIs(t, err, errCheck)
	assert.Equal(t, 1, port.calls)
}
