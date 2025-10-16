# TSL Pipeline Examples

This directory contains example configurations for the go-trust TSL pipeline processing system.

## YAML Configuration Format

Pipeline configurations are defined in YAML with a sequence of steps. Each step has a function name (like `load` or `transform`) and a list of arguments:

```yaml
# Example pipeline structure
- function-name:
    - argument1
    - argument2
    
- another-function:
    - argument
```

Note that there is **no `steps:` key** at the root level - the pipeline is a direct list of steps.

### Configuration Guidelines

IMPORTANT: Pipeline YAML files should:
- ONLY contain pipeline steps
- NOT include configuration such as `debug: true`
- NOT include any global configuration parameters

All configuration should be provided through command-line arguments when running the pipeline:

```bash
# Example: Enable debug mode through command line
gt --debug example/basic-usage.yaml
```

## Example Files

### 1. `basic-usage.yaml`

A simple example showing the core functionality:

- Loading a TSL from a URL
- Setting basic fetch options
- Creating a certificate pool
- Publishing the TSL to a local directory

### 2. `tsl-tree-publishing.yaml`

Demonstrates tree structure publishing with filtering:

- Loading a TSL with reference depth control
- Filtering by territory and service type
- Publishing with territory-based and index-based directory structures
- Creating filtered certificate pools

### 3. `api-and-html.yaml`

Shows integration with the API and HTML transformation:

- Loading TSLs for API access
- Transforming TSLs to HTML with XSLT
- Generating index pages
- Creating certificate pools with different filtering criteria

### 4. `custom-tsl-generation.yaml`

Demonstrates generating TSLs from metadata files:

- Creating TSLs from directory-based metadata
- Validating generated TSLs
- Optionally signing TSLs
- Publishing in different formats

## Using These Examples

To run these examples, use the gt pipeline command:

```bash
gt example/basic-usage.yaml
```

## Common Pipeline Steps

| Step Name | Description | Example Usage |
|-----------|-------------|--------------|
| `load` | Load a TSL from a URL or file | `- load: [https://example.com/tsl.xml]` |
| `set-fetch-options` | Configure fetch depth and filters | `- set-fetch-options: [max-depth:2, timeout:60s]` |
| `transform` | Apply an XSLT transformation | `- transform: [stylesheet.xslt, ./output, html]` |
| `publish` | Publish TSLs to a directory | `- publish: [./output]` |
| `log` | Log information | `- log: ["Loaded %d TSLs"]` |
| `select` | Extract certificates | `- select: [all]` |
| `generate` | Generate a TSL from metadata | `- generate: [./metadata-dir]` |
| `generate_index` | Create an index page | `- generate_index: [./output, "Title"]` |

## Tree Structure Publishing

When publishing, you can maintain the tree structure using these formats:

```yaml
# Territory-based directories
- publish:
    - ./output
    - tree:territory

# Index-based directories
- publish:
    - ./output
    - tree:index
```

## Certificate Selection and Filtering

```yaml
# Select all certificates
- select:
    - all

# Filter by service type
- select:
    - service-type:http://uri.etsi.org/TrstSvc/Svctype/CA/QC
    
# Filter by status with AND logic
- select:
    - status-logic:and
    - status:http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted
```