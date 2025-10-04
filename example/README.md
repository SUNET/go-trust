````markdown
# Examples

This directory contains examples for using the go-trust library.

## XSLT Stylesheet for TSL Visualization

The `tsl-to-html.xslt` file is an XSLT stylesheet that transforms ETSI TS 119 612 Trust Status Lists (TSLs) into comprehensive, user-friendly HTML documents. This makes it easy to visualize and navigate complex TSL data.

### Using the Stylesheet

You can use this stylesheet with any standard XSLT processor:

```bash
# Using xsltproc
xsltproc tsl-to-html.xslt path/to/tsl.xml > tsl.html

# Using Saxon
java -jar saxon.jar -s:path/to/tsl.xml -xsl:tsl-to-html.xslt -o:tsl.html
```

### Using with the Pipeline

This stylesheet is also available as an embedded resource in the Go-Trust binary. You can use it in two ways:

#### External File (this directory)
```yaml
- transform:
- path/to/example/tsl-to-html.xslt
- /output/directory
- html
```

#### Embedded File (no external dependencies)
```yaml
- transform:
- embedded:tsl-to-html.xslt
- /output/directory
- html
```

See the `embedded-transform.yaml` file in this directory for a complete example pipeline using the embedded stylesheet.

#### Programmatic Usage
```go
// Example code for transforming a TSL using the XSLT stylesheet
func TransformTSL(tslPath, outputPath string) error {
    p := pipeline.NewPipeline()
    p.AddStep(pipeline.NewLoadStep(tslPath))
    
    // Use the embedded stylesheet
    p.AddStep(pipeline.NewTransformStep("embedded:tsl-to-html.xslt", outputPath, "html"))
    
    return p.Execute()
}
```

## example-tsl

The `example-tsl` directory contains an example of a Trust Service List (TSL) directory structure that can be used with the `generate` pipeline step. This demonstrates how to organize trust service providers, their certificates, and metadata.

Structure:
- `scheme.yaml`: Contains the TSL scheme metadata (operator names, TSL type, etc.)
- `providers/`: Directory containing trust service providers
  - `example-provider/`: A sample trust service provider
    - `provider.yaml`: Provider metadata
    - `example.pem`: A certificate in PEM format
    - `example.yaml`: Certificate metadata

You can use this example with the pipeline as follows:

```yaml
- generate: ["/path/to/go-trust/example/example-tsl"]
- select: []  # Create a certificate pool from the generated TSL
- publish: ["/path/to/output"]  # Optional: Export the TSL as XML
```

This will:
1. Generate a TSL from the example directory structure
2. Create a certificate pool for validation
3. Export the TSL as XML files to the specified output directory