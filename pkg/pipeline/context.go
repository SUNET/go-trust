package pipeline

import (
	"crypto/x509"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"github.com/SUNET/go-trust/pkg/utils"
)

// Context holds the shared state passed between pipeline steps during processing.
// It contains Trust Status Lists (TSLs) and certificate pools that are created,
// modified, and consumed by different pipeline steps.
type Context struct {
	TSLs     *utils.Stack[*etsi119612.TSL] // A stack of Trust Status Lists being processed
	CertPool *x509.CertPool                // Certificate pool for trust verification
}

// EnsureTSLStack ensures that the TSL stack is initialized.
// If the stack doesn't exist, it creates a new one.
//
// This method is used by pipeline steps to guarantee that the TSL stack
// is available before operating on it, preventing nil pointer exceptions.
//
// Returns:
//   - The Context itself for method chaining
func (ctx *Context) EnsureTSLStack() *Context {
	if ctx.TSLs == nil {
		ctx.TSLs = utils.NewStack[*etsi119612.TSL]()
	}
	return ctx
}

// InitCertPool creates a new certificate pool in the context.
// This replaces any existing certificate pool with a fresh, empty one.
//
// This method is typically called before adding trusted certificates
// from Trust Status Lists to build a new trust store.
//
// Returns:
//   - The Context itself for method chaining
func (ctx *Context) InitCertPool() *Context {
	ctx.CertPool = x509.NewCertPool()
	return ctx
}

// NewContext creates a new pipeline context with initialized fields.
// The returned Context has a pre-initialized TSL stack ready to use,
// but no certificate pool (which should be created with InitCertPool when needed).
//
// Returns:
//   - A new Context instance with initialized TSL stack
func NewContext() *Context {
	return &Context{
		TSLs: utils.NewStack[*etsi119612.TSL](),
	}
}
