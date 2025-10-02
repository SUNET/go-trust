package pipeline

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

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
