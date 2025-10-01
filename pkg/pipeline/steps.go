package pipeline

import (
	"crypto/x509"
	"fmt"

	"github.com/SUNET/g119612/pkg/etsi119612"
)

// @PipelineStep("select")
func selectCertPool(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(ctx.TSLs) == 0 {
		return ctx, fmt.Errorf("select: no TSLs loaded in context")
	}
	mergedPool := x509.NewCertPool()
	var policy *etsi119612.TSPServicePolicy
	if len(args) == 0 {
		policy = etsi119612.PolicyAll
	} else {
		policy = etsi119612.NewTSPServicePolicy()
		for _, arg := range args {
			policy.AddServiceTypeIdentifier(arg)
		}
	}
	for _, tsl := range ctx.TSLs {
		if tsl == nil {
			continue
		}
		tsl.WithTrustServices(func(tsp *etsi119612.TSPType, svc *etsi119612.TSPServiceType) {
			svc.WithCertificates(func(cert *x509.Certificate) {
				if tsp.Validate(svc, []*x509.Certificate{cert}, policy) == nil {
					mergedPool.AddCert(cert)
				}
			})
		})
	}
	ctx.CertPool = mergedPool
	fmt.Printf("CertPool created from %d TSL(s) using policy (args: %v)\n", len(ctx.TSLs), args)
	return ctx, nil
}

// @PipelineStep("load")
func loadTSL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) == 0 {
		return ctx, fmt.Errorf("load: at least one TSL URL must be provided")
	}
	var tsls []*etsi119612.TSL
	for _, url := range args {
		tsl, err := etsi119612.FetchTSL(url)
		if err != nil {
			return ctx, fmt.Errorf("load: failed to fetch TSL from %s: %w", url, err)
		}
		tsls = append(tsls, tsl)
	}
	ctx.TSLs = tsls
	fmt.Printf("Loaded %d TSL(s)\n", len(tsls))
	return ctx, nil
}

// @PipelineStep("echo")
func echo(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	fmt.Println("echo:", args)
	return ctx, nil
}

// Function type for pipeline steps: Pipeline, Context, args; returns new Context and error
type PipeFunc func(pl *Pipeline, ctx *Context, args ...string) (*Context, error)

// Internal registry for mapping methodName to Go functions
var functionRegistry = make(map[string]PipeFunc)

// RegisterFunction registers a Go function with a methodName
func RegisterFunction(name string, fn PipeFunc) {
	functionRegistry[name] = fn
}

// GetFunctionByName retrieves a registered function by methodName
func GetFunctionByName(name string) (PipeFunc, bool) {
	fn, ok := functionRegistry[name]
	return fn, ok
}

// Register the pipeline step functions
func init() {
	RegisterFunction("echo", echo)
	RegisterFunction("load", loadTSL)
	RegisterFunction("select", selectCertPool)
}
