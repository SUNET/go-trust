// Package main provides the Go-Trust trust engine application entrypoint.
//
// Go-Trust is a local trust engine that provides trust decisions based on
// ETSI TS 119612 Trust Status Lists (TSLs). It allows clients to abstract trust
// decisions through an AuthZEN policy decision point (PDP). The service evaluates
// trust in subjects identified by X509 certificates using a set of TSLs.
//
// # Pipeline Overview
//
// Go-Trust processes TSLs using a YAML-defined pipeline. The pipeline consists of
// a series of steps, each performing specific operations on the TSLs. The YAML format
// is a sequence of pipeline steps, where each step has a name and a list of arguments.
//
// Example pipeline YAML format:
//
//	- load:
//	  - https://example.com/tsl.xml
//	- transform:
//	  - /path/to/stylesheet.xslt
//	  - replace
//	- publish:
//	  - /path/to/output
//
// # Available Pipeline Steps
//
// The following pipeline steps are available:
//
// - [pipeline.LoadTSL]: Loads a TSL from a URL or file path
//   - Args: URL or file path to load the TSL from
//
// - [pipeline.SelectCertPool]: Builds a certificate pool from the loaded TSLs
//   - Args: Service type URI filter (optional)
//
// - [pipeline.Echo]: Outputs debug information about the current pipeline context
//   - Args: Message prefix (optional)
//
// - [pipeline.GenerateTSL]: Generates a new TSL based on provided metadata
//   - Args: Path to scheme metadata YAML file, path to certificate metadata YAML file(s)
//
// - [pipeline.PublishTSL]: Serializes TSLs to XML files in a directory
//   - Args: Output directory path, optional signer configuration
//
// - [pipeline.TransformTSL]: Applies XSLT transformation to TSLs
//   - Args: XSLT stylesheet path, mode ("replace" or output directory), extension (optional)
//
// # Running the Application
//
// The application starts an API server that periodically processes the pipeline
// and provides endpoints for making trust decisions based on the processed TSLs.
//
// Command line options:
//
//	--host         API server hostname (default: 127.0.0.1)
//	--port         API server port (default: 6001)
//	--frequency    Pipeline update frequency (default: 5m)
//	--version      Show version information
//	--help         Show help message
//
// # API Endpoints
//
// The API server provides the following endpoints:
//
//	GET /status             - Get server status and TSL count information
//	                          Returns: {"tsl_count": <number>, "last_processed": <timestamp>}
//
//	GET /info               - Get detailed information about all loaded TSLs
//	                          Returns: {"tsl_summaries": [<tsl_summary>, ...]}
//
//	POST /authzen/decision  - Make an authorization decision using AuthZEN protocol
//	                          Accepts: AuthZEN EvaluationRequest with x5c certificate chains
//	                          Returns: AuthZEN EvaluationResponse with trust decision
//
// See: https://github.com/SUNET/go-trust for more information
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/SUNET/go-trust/pkg/api"
	"github.com/SUNET/go-trust/pkg/pipeline"
	"github.com/gin-gonic/gin"
)

// Version is set at build time using -ldflags
// It represents the current version of the Go-Trust application.
// Default value is "dev" for development builds. In production,
// this should be set to a specific version string using ldflags:
// go build -ldflags "-X main.Version=1.0.0" ./cmd
var Version = "dev"

// usage prints the command-line usage information to stderr.
// It shows the available command-line options and their descriptions.
func usage() {
	prog := os.Args[0]
	fmt.Fprintf(os.Stderr, "\nUsage: %s [options] <pipeline.yaml>\n", prog)
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  --help         Show this help message and exit.")
	fmt.Fprintln(os.Stderr, "  --version      Show version information and exit.")
	fmt.Fprintln(os.Stderr, "  --host         API server hostname (default: 127.0.0.1)")
	fmt.Fprintln(os.Stderr, "  --port         API server port (default: 6001)")
	fmt.Fprintln(os.Stderr, "  --frequency    Pipeline update frequency (default: 5m)")
	fmt.Fprintln(os.Stderr, "")
}

// main is the entry point for the Go-Trust application.
//
// It performs the following operations:
// 1. Parses command-line arguments and options
// 2. Loads the specified pipeline YAML file
// 3. Initializes the server context
// 4. Starts a background updater to periodically process the pipeline
// 5. Sets up the HTTP API server with Gin
// 6. Starts the API server on the specified address and port
//
// The pipeline YAML file defines the steps to process Trust Status Lists (TSLs).
// The processed TSLs are used by the API server to make trust decisions.
// See the [pipeline.Pipeline] documentation for details on the pipeline format.
func main() {
	showHelp := flag.Bool("help", false, "Show help message")
	showVersion := flag.Bool("version", false, "Show version information")
	host := flag.String("host", "127.0.0.1", "API server hostname")
	port := flag.String("port", "6001", "API server port")
	freq := flag.Duration("frequency", 5*time.Minute, "Pipeline update frequency (e.g. 10s, 1m, 5m)")
	flag.Parse()

	if *showHelp {
		usage()
		os.Exit(0)
	}
	if *showVersion {
		fmt.Println("Version:", Version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing pipeline YAML file argument.")
		usage()
		os.Exit(1)
	}

	pipelineFile := args[0]
	pl, err := pipeline.NewPipeline(pipelineFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load pipeline: %v\n", err)
		os.Exit(1)
	}

	serverCtx := &api.ServerContext{
		PipelineContext: &pipeline.Context{},
	}

	// Start background updater
	api.StartBackgroundUpdater(pl, serverCtx, *freq)

	// Gin API server
	r := gin.Default()
	api.RegisterAPIRoutes(r, serverCtx)
	listenAddr := fmt.Sprintf("%s:%s", *host, *port)
	fmt.Printf("API server listening on %s\n", listenAddr)
	if err := r.Run(listenAddr); err != nil {
		fmt.Fprintf(os.Stderr, "API server error: %v\n", err)
		os.Exit(1)
	}
}
