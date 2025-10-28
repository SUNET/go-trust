package api

import (
	"fmt"
	"os"
	"time"

	"crypto/x509"
	"github.com/SUNET/go-trust/pkg/authzen"
	"github.com/SUNET/go-trust/pkg/logging"
	"github.com/gin-gonic/gin"
)

// StatusHandler godoc
// @Summary Get server status
// @Description Returns the current server status including TSL count and last processing time
// @Tags Status
// @Produce json
// @Success 200 {object} map[string]interface{} "tsl_count, last_processed"
// @Router /status [get]
func StatusHandler(serverCtx *ServerContext) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// AuthZENDecisionHandler godoc
// @Summary Evaluate trust decision (AuthZEN)
// @Description Evaluates whether an X.509 certificate chain is trusted according to loaded TSLs
// @Description
// @Description This endpoint implements the AuthZEN evaluation protocol. It accepts certificate chains
// @Description in the x5c format (base64-encoded DER certificates) and validates them against the
// @Description trusted certificates loaded from ETSI TS 119612 Trust Status Lists.
// @Tags AuthZEN
// @Accept json
// @Produce json
// @Param request body authzen.EvaluationRequest true "AuthZEN Evaluation Request"
// @Success 200 {object} authzen.EvaluationResponse "Trust decision (decision=true for trusted, false for untrusted)"
// @Failure 400 {object} map[string]string "Invalid request format"
// @Router /authzen/decision [post]
func AuthZENDecisionHandler(serverCtx *ServerContext) gin.HandlerFunc {
	return func(c *gin.Context) {
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
				start := time.Now()
				opts := x509.VerifyOptions{
					Roots: certPool,
				}
				_, err := allCerts[0].Verify(opts)
				validationDuration := time.Since(start)

				if err == nil {
					serverCtx.Logger.Info("AuthZEN request approved",
						logging.F("remote_ip", c.ClientIP()),
						logging.F("subject", req.Subject.ID))

					// Record successful validation metrics
					if serverCtx.Metrics != nil {
						serverCtx.Metrics.RecordCertValidation(validationDuration, true)
					}

					c.JSON(200, buildResponse(true, ""))
				} else {
					serverCtx.Logger.Info("AuthZEN request denied",
						logging.F("remote_ip", c.ClientIP()),
						logging.F("subject", req.Subject.ID),
						logging.F("error", err.Error()))

					// Record failed validation metrics
					if serverCtx.Metrics != nil {
						serverCtx.Metrics.RecordCertValidation(validationDuration, false)
					}

					c.JSON(200, buildResponse(false, err.Error()))
				}
				return
			} else {
				serverCtx.Logger.Error("AuthZEN request failed - CertPool is nil",
					logging.F("remote_ip", c.ClientIP()))

				// Record error metrics
				if serverCtx.Metrics != nil {
					serverCtx.Metrics.RecordError("certpool_nil", "authzen_decision")
				}

				c.JSON(200, buildResponse(false, "CertPool is nil"))
				return
			}
		}
	}
}

// InfoHandler godoc
// @Summary Get TSL information
// @Description Returns detailed summaries of all loaded Trust Status Lists
// @Description
// @Description This endpoint provides comprehensive information about each TSL including:
// @Description - Territory code
// @Description - Sequence number
// @Description - Issue date
// @Description - Next update date
// @Description - Number of services
// @Tags Status
// @Produce json
// @Success 200 {object} map[string]interface{} "tsl_summaries"
// @Router /info [get]
func InfoHandler(serverCtx *ServerContext) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// TestShutdownHandler godoc (test mode only)
func TestShutdownHandler(serverCtx *ServerContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverCtx.Logger.Info("Shutdown requested via /test/shutdown endpoint",
			logging.F("remote_ip", c.ClientIP()))

		c.JSON(200, gin.H{"message": "shutting down"})

		// Trigger graceful shutdown after response is sent
		go func() {
			time.Sleep(100 * time.Millisecond) // Give time for response to be sent
			os.Exit(0)
		}()
	}
}
