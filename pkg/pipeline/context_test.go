package pipeline

import (
	"crypto/x509"
	"testing"

	"github.com/SUNET/go-trust/pkg/logging"
	etsi119612 "github.com/SUNET/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithLogger(t *testing.T) {
	t.Run("Replace logger", func(t *testing.T) {
		// Create initial pipeline with default logger
		pl := &Pipeline{
			Pipes:  []Pipe{{MethodName: "test", MethodArguments: []string{}}},
			Logger: logging.NewLogger(logging.InfoLevel),
		}

		// Create new logger with different level
		debugLogger := logging.NewLogger(logging.DebugLevel)

		// Replace logger
		newPl := pl.WithLogger(debugLogger)

		// Verify new pipeline has new logger
		assert.NotNil(t, newPl)
		assert.Equal(t, debugLogger, newPl.Logger)

		// Verify pipes are preserved
		assert.Equal(t, pl.Pipes, newPl.Pipes)

		// Verify original pipeline unchanged
		assert.NotEqual(t, pl.Logger, newPl.Logger)
	})

	t.Run("Nil logger falls back to default", func(t *testing.T) {
		pl := &Pipeline{
			Pipes: []Pipe{{MethodName: "test", MethodArguments: []string{}}},
		}

		newPl := pl.WithLogger(nil)

		assert.NotNil(t, newPl)
		assert.NotNil(t, newPl.Logger)
	})

	t.Run("Preserves pipes", func(t *testing.T) {
		pipes := []Pipe{
			{MethodName: "load", MethodArguments: []string{"url"}},
			{MethodName: "transform", MethodArguments: []string{"xslt"}},
		}

		pl := &Pipeline{
			Pipes:  pipes,
			Logger: logging.NewLogger(logging.InfoLevel),
		}

		newLogger := logging.NewLogger(logging.DebugLevel)
		newPl := pl.WithLogger(newLogger)

		require.NotNil(t, newPl)
		assert.Equal(t, len(pipes), len(newPl.Pipes))
		assert.Equal(t, "load", newPl.Pipes[0].MethodName)
		assert.Equal(t, "transform", newPl.Pipes[1].MethodName)
	})
}

func TestAddTSL_EdgeCases(t *testing.T) {
	t.Run("Add nil TSL", func(t *testing.T) {
		ctx := NewContext()
		ctx.AddTSL(nil)

		// AddTSL returns early for nil, so nothing is added
		assert.Equal(t, 0, ctx.TSLs.Size())
		assert.Equal(t, 0, ctx.TSLTrees.Size())
	})

	t.Run("Add multiple TSLs", func(t *testing.T) {
		ctx := NewContext()

		// Add several TSLs
		// Note: AddTSL creates a tree and traverses it, which can add multiple TSLs to the stack
		for i := 0; i < 10; i++ {
			ctx.AddTSL(&etsi119612.TSL{})
		}

		// Each TSL gets added twice - once directly and once via tree traversal
		assert.Equal(t, 20, ctx.TSLs.Size())
		assert.Equal(t, 10, ctx.TSLTrees.Size())
	})
}

func TestContext_Copy_DeepCopy(t *testing.T) {
	t.Run("Modifications don't affect original", func(t *testing.T) {
		original := NewContext()
		original.Data["key1"] = "value1"
		original.AddTSL(&etsi119612.TSL{})

		// Original has 1 data entry and 2 TSLs (AddTSL adds via both tree and direct push)
		assert.Equal(t, 1, len(original.Data))
		originalTSLCount := original.TSLs.Size()

		// Make a copy
		copied := original.Copy()

		// Modify the copy
		copied.Data["key2"] = "value2"
		copied.AddTSL(&etsi119612.TSL{})

		// Verify original unchanged
		assert.Equal(t, 1, len(original.Data))
		assert.Equal(t, originalTSLCount, original.TSLs.Size())

		// Verify copy has modifications
		assert.Equal(t, 2, len(copied.Data))
		assert.Greater(t, copied.TSLs.Size(), originalTSLCount)
	})

	t.Run("Copy with CertPool", func(t *testing.T) {
		original := NewContext()
		original.CertPool = x509.NewCertPool()

		copied := original.Copy()

		assert.NotNil(t, copied.CertPool)
		// CertPool is recreated (not the same reference)
		assert.NotSame(t, original.CertPool, copied.CertPool)
	})
}
