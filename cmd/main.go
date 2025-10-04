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
//   - load:
//   - https://example.com/tsl.xml
//   - transform:
//   - /path/to/stylesheet.xslt
//   - replace
//   - publish:
//   - /path/to/output
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
// Logging options:
//
//	--log-level    Logging level: debug, info, warn, error, fatal (default: info)
//	--log-format   Logging format: text or json (default: text)
//	--log-output   Log output: stdout, stderr, or file path (default: stdout)
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
	"path/filepath"
	"strings"
	"time"

	"github.com/SUNET/go-trust/pkg/api"
	"github.com/SUNET/go-trust/pkg/logging"
	"github.com/SUNET/go-trust/pkg/pipeline"
	"github.com/gin-gonic/gin"
)

// Version is set at build time using -ldflags
// It represents the current version of the Go-Trust application.
// Default value is "dev" for development builds. In production,
// this should be set to a specific version string using ldflags:
// go build -ldflags "-X main.Version=1.0.0" ./cmd
var Version = "dev"

// parseLogLevel converts a string log level to the corresponding LogLevel enum value.
// This is used to convert command-line arguments to the internal level representation.
//
// Valid values are:
//   - "debug": Detailed debugging information (most verbose)
//   - "info": Normal operation messages (default)
//   - "warn" or "warning": Warning conditions
//   - "error": Error conditions
//   - "fatal": Critical conditions that require application termination
//
// If an invalid or unknown level is provided, the function returns InfoLevel
// with a warning message printed to stderr.
func parseLogLevel(level string) logging.LogLevel {
	level = strings.ToLower(level)
	switch level {
	case "debug":
		return logging.DebugLevel
	case "info":
		return logging.InfoLevel
	case "warn", "warning":
		return logging.WarnLevel
	case "error":
		return logging.ErrorLevel
	case "fatal":
		return logging.FatalLevel
	default:
		fmt.Fprintf(os.Stderr, "Warning: unknown log level '%s', using 'info'\n", level)
		return logging.InfoLevel
	}
}

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
	fmt.Fprintln(os.Stderr, "Logging options:")
	fmt.Fprintln(os.Stderr, "  --log-level    Logging level: debug, info, warn, error, fatal (default: info)")
	fmt.Fprintln(os.Stderr, "  --log-format   Logging format: text or json (default: text)")
	fmt.Fprintln(os.Stderr, "  --log-output   Log output: stdout, stderr, or file path (default: stdout)")
	fmt.Fprintln(os.Stderr, "")
}

// main is the entry point for the Go-Trust application.
//
// It performs the following operations:
// 1. Parses command-line arguments and options (including logging options)
// 2. Configures structured logging based on command line arguments
// 3. Loads the specified pipeline YAML file
// 4. Initializes the server context with configured logger
// 5. Starts a background updater to periodically process the pipeline
// 6. Sets up the HTTP API server with Gin
// 7. Starts the API server on the specified address and port
//
// The pipeline YAML file defines the steps to process Trust Status Lists (TSLs).
// The processed TSLs are used by the API server to make trust decisions.
// See the [pipeline.Pipeline] documentation for details on the pipeline format.
//
// Logging is configured using the --log-level, --log-format, and --log-output flags.
// The log level determines which message severity levels are output (debug, info, warn, error, fatal).
// The log format can be either human-readable text or structured JSON for machine processing.
// Log output can be directed to stdout, stderr, or a file path.
func main() {
	showHelp := flag.Bool("help", false, "Show help message")
	showVersion := flag.Bool("version", false, "Show version information")
	host := flag.String("host", "127.0.0.1", "API server hostname")
	port := flag.String("port", "6001", "API server port")
	freq := flag.Duration("frequency", 5*time.Minute, "Pipeline update frequency (e.g. 10s, 1m, 5m)")

	// Logging configuration
	logLevel := flag.String("log-level", "info", "Logging level (debug, info, warn, error, fatal)")
	logFormat := flag.String("log-format", "text", "Logging format (text, json)")
	logOutput := flag.String("log-output", "stdout", "Log output (stdout, stderr, or file path)")

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

	// Configure logger based on command line arguments
	parsedLogLevel := parseLogLevel(*logLevel)
	var logger logging.Logger

	if strings.ToLower(*logFormat) == "json" {
		logger = logging.JSONLogger(parsedLogLevel)
	} else {
		logger = logging.NewLogger(parsedLogLevel)
	}

	// Configure log output
	output := strings.ToLower(*logOutput)
	switch output {
	case "stdout":
		// Default is already stdout
	case "stderr":
		logger.(logging.OutputConfigurable).SetOutput(os.Stderr)
	default:
		// Assume it's a file path
		dir := filepath.Dir(*logOutput)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
			os.Exit(1)
		}

		file, err := os.OpenFile(*logOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		logger.(logging.OutputConfigurable).SetOutput(file)
	}

	// Configure pipeline with logger
	pl, err := pipeline.NewPipeline(pipelineFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load pipeline: %v\n", err)
		os.Exit(1)
	}
	// Create a pipeline with our configured logger
	pl = pl.WithLogger(logger)

	// Create server context with logger
	serverCtx := api.NewServerContext(logger)
	serverCtx.PipelineContext = &pipeline.Context{}

	// Start background updater
	api.StartBackgroundUpdater(pl, serverCtx, *freq)

	// Gin API server
	r := gin.Default()
	api.RegisterAPIRoutes(r, serverCtx)
	listenAddr := fmt.Sprintf("%s:%s", *host, *port)

	// Log startup information
	logger.Info("API server starting",
		logging.F("address", listenAddr),
		logging.F("version", Version),
		logging.F("pipeline", pipelineFile),
		logging.F("log_level", *logLevel))

	if err := r.Run(listenAddr); err != nil {
		logger.Error("API server failed to start",
			logging.F("error", err.Error()),
			logging.F("address", listenAddr))
		os.Exit(1)
	}
}
