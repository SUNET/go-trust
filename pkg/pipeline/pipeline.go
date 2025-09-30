package pipeline

import (
	"crypto/x509"
	"fmt"
	"os"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"gopkg.in/yaml.v3"
)

// Context holds state passed between pipeline steps
type Context struct {
	TSLs     []*etsi119612.TSL
	CertPool *x509.CertPool
}

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

// Annotation pattern: use a comment like '// @PipelineStep("methodName")' above the function

// Example:
// @PipelineStep("echo")
func echo(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	fmt.Println("echo:", args)
	return ctx, nil
}

// Register the example function (in init)
func init() {
	RegisterFunction("echo", echo)
	RegisterFunction("load", loadTSL)
	RegisterFunction("select", selectCertPool)
}

// Pipeline represents a list of Pipe steps
type Pipeline struct {
	Pipes []Pipe
}

// Process iterates over the pipeline steps, passing and returning Context
func (pl *Pipeline) Process(ctx *Context) (*Context, error) {
	for i, pipe := range pl.Pipes {
		fn, ok := GetFunctionByName(pipe.MethodName)
		if !ok {
			return nil, fmt.Errorf("step %d: unknown methodName '%s'", i, pipe.MethodName)
		}
		var err error
		ctx, err = fn(pl, ctx, pipe.MethodArguments...)
		if err != nil {
			return ctx, fmt.Errorf("step %d (%s) failed: %w", i, pipe.MethodName, err)
		}
	}
	return ctx, nil
}

// NewPipeline loads a YAML file and returns a Pipeline instance
func NewPipeline(filename string) (*Pipeline, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var pipes []Pipe
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&pipes); err != nil {
		return nil, err
	}

	return &Pipeline{Pipes: pipes}, nil
}

// Pipe represents a method and its arguments in the pipeline

// Pipe represents a method and its arguments in the pipeline, with custom unmarshalling
type Pipe struct {
	MethodName      string
	MethodArguments []string
}

// UnmarshalYAML implements custom unmarshalling for Pipe
func (p *Pipe) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode || len(value.Content) != 2 {
		return &yaml.TypeError{Errors: []string{"Pipe must be a map with a single key (method name) and a list of arguments"}}
	}
	methodNode := value.Content[0]
	argsNode := value.Content[1]
	p.MethodName = methodNode.Value
	if argsNode.Kind != yaml.SequenceNode {
		return &yaml.TypeError{Errors: []string{"Pipe arguments must be a sequence"}}
	}
	p.MethodArguments = make([]string, len(argsNode.Content))
	for i, arg := range argsNode.Content {
		p.MethodArguments[i] = arg.Value
	}
	return nil
}
