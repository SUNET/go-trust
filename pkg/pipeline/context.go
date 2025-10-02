package pipeline

import (
	"crypto/x509"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"github.com/SUNET/go-trust/pkg/utils"
)

// Context holds state passed between pipeline steps
type Context struct {
	TSLs     *utils.Stack[*etsi119612.TSL]
	CertPool *x509.CertPool
}

// EnsureTSLStack ensures that the TSL stack is initialized.
// If the stack doesn't exist, it creates a new one.
func (ctx *Context) EnsureTSLStack() *Context {
	if ctx.TSLs == nil {
		ctx.TSLs = utils.NewStack[*etsi119612.TSL]()
	}
	return ctx
}

// InitCertPool creates a new certificate pool in the context.
// This replaces any existing certificate pool.
func (ctx *Context) InitCertPool() *Context {
	ctx.CertPool = x509.NewCertPool()
	return ctx
}

// NewContext creates a new pipeline context with initialized fields
func NewContext() *Context {
	return &Context{
		TSLs: utils.NewStack[*etsi119612.TSL](),
	}
}
