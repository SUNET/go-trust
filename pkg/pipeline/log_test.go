package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	ctx := NewContext()

	// Test with no arguments
	resultCtx, err := Log(nil, ctx)
	assert.NoError(t, err)
	assert.Equal(t, ctx, resultCtx)

	// Test with a message argument
	resultCtx, err = Log(nil, ctx, "Test log message")
	assert.NoError(t, err)
	assert.Equal(t, ctx, resultCtx)

	// The Log function should not modify the context
	assert.Equal(t, ctx, resultCtx)
}
