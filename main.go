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
var Version = "dev"

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
