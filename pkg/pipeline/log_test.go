package pipeline

import (
	"testing"

	"github.com/SUNET/go-trust/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	ctx := NewContext()
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test with no arguments
	resultCtx, err := Log(pl, ctx)
	assert.NoError(t, err)
	assert.Equal(t, ctx, resultCtx)

	// Test with a message argument
	resultCtx, err = Log(pl, ctx, "Test log message")
	assert.NoError(t, err)
	assert.Equal(t, ctx, resultCtx)

	// The Log function should not modify the context
	assert.Equal(t, ctx, resultCtx)
}
