# Examples

This directory contains examples for using the go-trust library.

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