package api

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/SUNET/go-trust/pkg/authzen"
	"github.com/SUNET/go-trust/pkg/logging"
	"github.com/SUNET/go-trust/pkg/pipeline"
	"github.com/gin-gonic/gin"
)

// parseX5C extracts and parses x5c certificates from a map[string]interface{}.
func parseX5C(props map[string]interface{}) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	if props == nil {
		return certs, nil
	}
	x5cVal, ok := props["x5c"]
	if !ok {
		return certs, nil
	}
	x5cList, ok := x5cVal.([]interface{})
	if !ok {
		return certs, fmt.Errorf("x5c property is not a list")
	}
	for _, item := range x5cList {
		str, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("x5c entry is not a string")
		}
		der, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode x5c entry: %v", err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("failed to parse x5c certificate: %v", err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// buildResponse constructs an EvaluationResponse for the AuthZEN API.
func buildResponse(decision bool, reason string) authzen.EvaluationResponse {
	if decision {
		return authzen.EvaluationResponse{Decision: true}
	}
	return authzen.EvaluationResponse{
		Decision: false,
		Context: &struct {
			ID          string                 `json:"id"`
			ReasonAdmin map[string]interface{} `json:"reason_admin,omitempty"`
			ReasonUser  map[string]interface{} `json:"reason_user,omitempty"`
		}{
			ReasonAdmin: map[string]interface{}{"error": reason},
		},
	}
}

// StartBackgroundUpdater runs the pipeline at regular intervals and updates the server context.
// This function starts a goroutine that processes the pipeline at the specified frequency
// and updates the ServerContext with the new pipeline results. The updated context is then
// used by API handlers to respond to requests with fresh data.
//
// The pipeline is processed immediately upon calling this function, before starting the
// background updates. This ensures TSLs are available as soon as the server starts.
//
// Success and failure events are logged using the ServerContext's structured logger:
// - On success: An info-level message with the update frequency
// - On failure: An error-level message with the error details and frequency
//
// Parameters:
//   - pl: The pipeline to process periodically
//   - serverCtx: The server context to update with pipeline results (must have a valid logger)
//   - freq: The frequency at which to process the pipeline (e.g., 5m for every 5 minutes)
//
// This function is typically called at server startup to ensure TSLs are kept up-to-date.
func StartBackgroundUpdater(pl *pipeline.Pipeline, serverCtx *ServerContext, freq time.Duration) error {
	// Process pipeline immediately to ensure TSLs are loaded without waiting
	newCtx, err := pl.Process(pipeline.NewContext())
	serverCtx.Lock()
	if err == nil && newCtx != nil {
		serverCtx.PipelineContext = newCtx
		serverCtx.LastProcessed = time.Now()
		tslCount := countTSLs(newCtx)
		serverCtx.Logger.Info("Initial pipeline processing successful",
			logging.F("tsl_count", tslCount))
	} else if err != nil {
		serverCtx.Logger.Error("Initial pipeline processing failed",
			logging.F("error", err.Error()))
	}
	serverCtx.Unlock()

	// Start background processing
	go func() {
		for {
			time.Sleep(freq)

			newCtx, err := pl.Process(pipeline.NewContext())
			serverCtx.Lock()
			if err == nil && newCtx != nil {
				serverCtx.PipelineContext = newCtx
				serverCtx.LastProcessed = time.Now()
			}
			serverCtx.Unlock()
			if err != nil {
				// ServerContext always has a logger after our improvements
				serverCtx.Logger.Error("Pipeline processing failed",
					logging.F("error", err.Error()),
					logging.F("frequency", freq.String()))
			} else {
				// Log successful update
				tslCount := countTSLs(newCtx)
				serverCtx.Logger.Info("Pipeline processed successfully",
					logging.F("frequency", freq.String()),
					logging.F("tsl_count", tslCount))
			}
		}
	}()
	return nil
}

// countTSLs counts the number of TSLs in the pipeline context.
// This is a helper function to provide consistent TSL counting for logging.
func countTSLs(ctx *pipeline.Context) int {
	if ctx == nil || ctx.TSLs == nil {
		return 0
	}
	return ctx.TSLs.Size()
}

// NewServerContext creates a new ServerContext with a configured logger.
// The ServerContext will always have a valid logger - if none is provided,
// it will use the DefaultLogger.
func NewServerContext(logger logging.Logger) *ServerContext {
	// Always ensure a valid logger
	if logger == nil {
		logger = logging.DefaultLogger()
	}
	return &ServerContext{
		Logger: logger,
	}
}

// RegisterAPIRoutes registers all API endpoints on the given Gin router using ServerContext.
// It sets up the following endpoints:
//
// GET /status - Returns the current server status including TSL count and last processing time
//
// GET /info - Returns detailed summaries of all TSLs in the current pipeline context
//
// POST /authzen/decision - Implements the AuthZEN protocol for making trust decisions
//
//	This endpoint processes AuthZEN EvaluationRequest objects containing x5c certificate
//	chains and verifies them against the trusted certificates in the pipeline context.
func RegisterAPIRoutes(r *gin.Engine, serverCtx *ServerContext) {
	// Status endpoint returns basic server status information
	// including the count of TSLs and when the pipeline was last processed
	r.GET("/status", func(c *gin.Context) {
		serverCtx.RLock()
		defer serverCtx.RUnlock()
		tslCount := 0
		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			tslCount = serverCtx.PipelineContext.TSLs.Size()
		}

		// Log the status request with structured logging
		serverCtx.Logger.Info("API status request",
			logging.F("remote_ip", c.ClientIP()),
			logging.F("tsl_count", tslCount))

		c.JSON(200, gin.H{
			"tsl_count":      tslCount,
			"last_processed": serverCtx.LastProcessed.Format("2006-01-02T15:04:05Z07:00"),
		})
	})

	// AuthZEN decision endpoint implements the AuthZEN protocol for making trust decisions
	// It processes AuthZEN EvaluationRequest objects containing x5c certificate chains
	// and verifies them against the trusted certificates in the pipeline context
	r.POST("/authzen/decision", func(c *gin.Context) {
		var req authzen.EvaluationRequest
		if err := c.BindJSON(&req); err != nil {
			// Log invalid request with structured logging
			serverCtx.Logger.Error("Invalid AuthZEN request",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("error", err.Error()))
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}

		// Log valid request
		serverCtx.Logger.Debug("Processing AuthZEN request",
			logging.F("remote_ip", c.ClientIP()))

		// Try to extract x5c from subject, resource, action, and context
		var allCerts []*x509.Certificate
		var parseErr error
		subjectCerts, err := parseX5C(req.Subject.Properties)
		if err != nil {
			parseErr = fmt.Errorf("subject x5c: %v", err)
		}
		allCerts = append(allCerts, subjectCerts...)
		resourceCerts, err := parseX5C(req.Resource.Properties)
		if err != nil && parseErr == nil {
			parseErr = fmt.Errorf("resource x5c: %v", err)
		}
		allCerts = append(allCerts, resourceCerts...)
		actionCerts, err := parseX5C(req.Action.Properties)
		if err != nil && parseErr == nil {
			parseErr = fmt.Errorf("action x5c: %v", err)
		}
		allCerts = append(allCerts, actionCerts...)
		contextCerts, err := parseX5C(req.Context)
		if err != nil && parseErr == nil {
			parseErr = fmt.Errorf("context x5c: %v", err)
		}
		allCerts = append(allCerts, contextCerts...)

		// If any x5c parse error, return error in AuthZEN-compatible response
		if parseErr != nil {
			serverCtx.Logger.Error("AuthZEN certificate parsing error",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("error", parseErr.Error()))
			c.JSON(200, buildResponse(false, parseErr.Error()))
			return
		}

		if len(allCerts) > 0 {
			serverCtx.RLock()
			certPool := serverCtx.PipelineContext.CertPool
			serverCtx.RUnlock()
			if certPool != nil {
				opts := x509.VerifyOptions{
					Roots: certPool,
				}
				_, err := allCerts[0].Verify(opts)
				if err == nil {
					serverCtx.Logger.Info("AuthZEN request approved",
						logging.F("remote_ip", c.ClientIP()),
						logging.F("subject", req.Subject.ID))
					c.JSON(200, buildResponse(true, ""))
				} else {
					serverCtx.Logger.Info("AuthZEN request denied",
						logging.F("remote_ip", c.ClientIP()),
						logging.F("subject", req.Subject.ID),
						logging.F("error", err.Error()))
					c.JSON(200, buildResponse(false, err.Error()))
				}
				return
			} else {
				serverCtx.Logger.Error("AuthZEN request failed - CertPool is nil",
					logging.F("remote_ip", c.ClientIP()))
				c.JSON(200, buildResponse(false, "CertPool is nil"))
				return
			}
		}
	})

	// Info endpoint returns detailed information about all loaded TSLs
	// It provides summaries of each TSL in the current pipeline context
	r.GET("/info", func(c *gin.Context) {
		serverCtx.RLock()
		defer serverCtx.RUnlock()
		summaries := make([]map[string]interface{}, 0)

		// Add debug logging to inspect the pipeline context
		tslSize := 0
		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			tslSize = serverCtx.PipelineContext.TSLs.Size()
		}

		serverCtx.Logger.Debug("API info request: Inspecting pipeline context",
			logging.F("ctx_nil", serverCtx.PipelineContext == nil),
			logging.F("tsls_nil", serverCtx.PipelineContext == nil || serverCtx.PipelineContext.TSLs == nil),
			logging.F("tsls_size", tslSize))

		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			for _, tsl := range serverCtx.PipelineContext.TSLs.ToSlice() {
				if tsl != nil {
					summaries = append(summaries, tsl.Summary())
				}
			}
		}

		// Log info request with structured logging
		serverCtx.Logger.Info("API info request",
			logging.F("remote_ip", c.ClientIP()),
			logging.F("summary_count", len(summaries)))

		c.JSON(200, gin.H{
			"tsl_summaries": summaries,
		})
	})

	// Test-mode shutdown endpoint
	// This endpoint is only registered when GO_TRUST_TEST_MODE environment variable is set
	// It allows integration tests to gracefully shutdown the server
	if os.Getenv("GO_TRUST_TEST_MODE") == "1" {
		r.POST("/test/shutdown", func(c *gin.Context) {
			serverCtx.Logger.Info("Shutdown requested via /test/shutdown endpoint",
				logging.F("remote_ip", c.ClientIP()))

			c.JSON(200, gin.H{"message": "shutting down"})

			// Trigger graceful shutdown after response is sent
			go func() {
				time.Sleep(100 * time.Millisecond) // Give time for response to be sent
				os.Exit(0)
			}()
		})

		serverCtx.Logger.Warn("Test mode enabled: /test/shutdown endpoint is available")
	}
}
