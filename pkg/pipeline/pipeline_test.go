package pipeline

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestPipeline_Process_Success(t *testing.T) {
	RegisterFunction("testfunc", func(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
		assert.Equal(t, []string{"foo", "bar"}, args)
		if ctx == nil {
			t.Fatal("ctx should not be nil")
		}
		ctx.TSLs = append(ctx.TSLs, nil) // simulate adding a TSL
		return ctx, nil
	})
	yamlData := `
- testfunc:
    - foo
    - bar
`
	var pipes []Pipe
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := &Pipeline{Pipes: pipes}
	ctx, err := pl.Process(&Context{})
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.Len(t, ctx.TSLs, 1)
}

func TestPipeline_Process_UnknownMethod(t *testing.T) {
	yamlData := `
- unknown:
    - foo
`
	var pipes []Pipe
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := &Pipeline{Pipes: pipes}
	ctx, err := pl.Process(&Context{})
	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "unknown methodName")
}

func TestPipeline_Process_FuncError(t *testing.T) {
	RegisterFunction("failfunc", func(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
		return ctx, os.ErrPermission
	})
	yamlData := `
- failfunc:
    - foo
`
	var pipes []Pipe
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := &Pipeline{Pipes: pipes}
	ctx, err := pl.Process(&Context{})
	assert.Error(t, err)
	assert.NotNil(t, ctx)
	assert.Contains(t, err.Error(), "failed")
}
