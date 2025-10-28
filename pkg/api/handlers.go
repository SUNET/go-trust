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
// @Summary Evaluate trust decision (AuthZEN Trust Registry Profile)
// @Description Evaluates whether a name-to-key binding is trusted according to loaded TSLs
// @Description
// @Description This endpoint implements the AuthZEN Trust Registry Profile as specified in
// @Description draft-johansson-authzen-trust. It validates that a public key (in resource.key)
// @Description is correctly bound to a name (in subject.id) according to ETSI TS 119612 Trust Status Lists.
// @Description
// @Description The request MUST have:
// @Description - subject.type = "key" and subject.id = the name to validate
// @Description - resource.type = "jwk" or "x5c" with resource.key containing the public key/certificates
// @Description - resource.id MUST equal subject.id
// @Description - action (optional) with name = the role being validated
// @Tags AuthZEN
// @Accept json
// @Produce json
// @Param request body authzen.EvaluationRequest true "AuthZEN Trust Registry Evaluation Request"
// @Success 200 {object} authzen.EvaluationResponse "Trust decision (decision=true for trusted, false for untrusted)"
// @Failure 400 {object} map[string]string "Invalid request format or validation error"
// @Router /evaluation [post]
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

		// Validate request against AuthZEN Trust Registry Profile
		if err := req.Validate(); err != nil {
			serverCtx.Logger.Error("AuthZEN request validation failed",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("error", err.Error()))
			c.JSON(400, gin.H{"error": fmt.Sprintf("validation error: %v", err)})
			return
		}

		// Log valid request
		serverCtx.Logger.Debug("Processing AuthZEN request",
			logging.F("remote_ip", c.ClientIP()),
			logging.F("subject_id", req.Subject.ID),
			logging.F("resource_type", req.Resource.Type))

		// Extract certificates from resource.key based on resource.type
		var certs []*x509.Certificate
		var parseErr error

		if req.Resource.Type == "x5c" {
			// resource.key is an array of base64-encoded X.509 certificates
			certs, parseErr = parseX5CFromArray(req.Resource.Key)
		} else {
			// resource.type == "jwk" - extract certificate from JWK x5c claim
			certs, parseErr = parseX5CFromJWK(req.Resource.Key)
		}

		if parseErr != nil {
			serverCtx.Logger.Error("AuthZEN certificate parsing error",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("resource_type", req.Resource.Type),
				logging.F("error", parseErr.Error()))
			c.JSON(200, buildResponse(false, parseErr.Error()))
			return
		}

		if len(certs) == 0 {
			serverCtx.Logger.Error("AuthZEN request has no certificates",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("resource_type", req.Resource.Type))
			c.JSON(200, buildResponse(false, "no certificates found in resource.key"))
			return
		}

		// Validate certificate chain against TSL certificate pool
		serverCtx.RLock()
		certPool := serverCtx.PipelineContext.CertPool
		serverCtx.RUnlock()

		if certPool == nil {
			serverCtx.Logger.Error("AuthZEN request failed - CertPool is nil",
				logging.F("remote_ip", c.ClientIP()))

			// Record error metrics
			if serverCtx.Metrics != nil {
				serverCtx.Metrics.RecordError("certpool_nil", "authzen_decision")
			}

			c.JSON(200, buildResponse(false, "CertPool is nil"))
			return
		}

		start := time.Now()
		opts := x509.VerifyOptions{
			Roots: certPool,
		}
		_, err := certs[0].Verify(opts)
		validationDuration := time.Since(start)

		if err == nil {
			serverCtx.Logger.Info("AuthZEN request approved",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("subject_id", req.Subject.ID),
				logging.F("resource_type", req.Resource.Type))

			// Record successful validation metrics
			if serverCtx.Metrics != nil {
				serverCtx.Metrics.RecordCertValidation(validationDuration, true)
			}

			c.JSON(200, buildResponse(true, ""))
		} else {
			serverCtx.Logger.Info("AuthZEN request denied",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("subject_id", req.Subject.ID),
				logging.F("resource_type", req.Resource.Type),
				logging.F("error", err.Error()))

			// Record failed validation metrics
			if serverCtx.Metrics != nil {
				serverCtx.Metrics.RecordCertValidation(validationDuration, false)
			}

			c.JSON(200, buildResponse(false, err.Error()))
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

// WellKnownHandler godoc
// @Summary AuthZEN PDP discovery endpoint
// @Description Returns Policy Decision Point metadata according to Section 9 of the AuthZEN specification
// @Description This endpoint provides service discovery information including supported endpoints and capabilities
// @Description per RFC 8615 well-known URI registration
// @Tags AuthZEN
// @Produce json
// @Success 200 {object} authzen.PDPMetadata "PDP metadata"
// @Router /.well-known/authzen-configuration [get]
func WellKnownHandler(baseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Construct metadata according to AuthZEN spec Section 9.1
		metadata := authzen.PDPMetadata{
			PolicyDecisionPoint:      baseURL,
			AccessEvaluationEndpoint: baseURL + "/evaluation",
			// Optional endpoints - not implemented yet
			// AccessEvaluationsEndpoint: baseURL + "/evaluations",
			// SearchSubjectEndpoint: baseURL + "/search/subject",
			// SearchResourceEndpoint: baseURL + "/search/resource",
			// SearchActionEndpoint: baseURL + "/search/action",
			// Capabilities: []string{}, // Could list custom capabilities here
		}

		// Return metadata with proper Content-Type
		c.JSON(200, metadata)
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
