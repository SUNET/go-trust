package api

import (
	"fmt"
	"os"
	"time"

	"crypto/x509"

	"github.com/SUNET/go-trust/pkg/authzen"
	"github.com/SUNET/go-trust/pkg/logging"
	"github.com/SUNET/go-trust/pkg/utils/x509util"
	"github.com/gin-gonic/gin"
)

// StatusHandler godoc
// @Summary Get server status (DEPRECATED - use GET /readyz)
// @Description Returns the current server status including TSL count and last processing time
// @Description
// @Description DEPRECATED: This endpoint is deprecated. Use GET /readyz for health checks.
// @Tags Status
// @Deprecated true
// @Produce json
// @Success 200 {object} map[string]interface{} "tsl_count, last_processed"
// @Router /status [get]
func StatusHandler(serverCtx *ServerContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add deprecation headers
		c.Header("Deprecation", "true")
		c.Header("Link", "</readyz>; rel=\"alternate\"")
		c.Header("X-API-Warn", "This endpoint is deprecated. Please use GET /readyz instead.")

		serverCtx.RLock()
		defer serverCtx.RUnlock()
		tslCount := 0
		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			tslCount = serverCtx.PipelineContext.TSLs.Size()
		}

		// Log the status request with structured logging
		serverCtx.Logger.Warn("API status request (deprecated endpoint)",
			logging.F("remote_ip", c.ClientIP()),
			logging.F("tsl_count", tslCount),
			logging.F("replacement", "GET /readyz"))

		c.JSON(200, gin.H{
			"tsl_count":      tslCount,
			"last_processed": serverCtx.LastProcessed.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
}

// AuthZENDecisionHandler godoc
// @Summary Evaluate trust decision (AuthZEN Trust Registry Profile)
// @Description Evaluates whether a name-to-key binding is trusted according to loaded trust registries
// @Description
// @Description This endpoint implements the AuthZEN Trust Registry Profile as specified in
// @Description draft-johansson-authzen-trust. It validates that a public key (in resource.key)
// @Description is correctly bound to a name (in subject.id) using configured trust registries
// @Description (ETSI TS 119612 TSLs, OpenID Federation, DID methods, etc.).
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

		// Log valid request
		serverCtx.Logger.Debug("Processing AuthZEN request",
			logging.F("remote_ip", c.ClientIP()),
			logging.F("subject_id", req.Subject.ID),
			logging.F("resource_type", req.Resource.Type))

		start := time.Now()

		// Use RegistryManager if available, fallback to legacy PipelineContext
		serverCtx.RLock()
		registryMgr := serverCtx.RegistryManager
		serverCtx.RUnlock()

		var resp *authzen.EvaluationResponse
		var evalErr error

		if registryMgr != nil {
			// New architecture: use RegistryManager
			resp, evalErr = registryMgr.Evaluate(c.Request.Context(), &req)
		} else {
			// Legacy architecture: use direct validation (backward compatibility)
			resp, evalErr = legacyEvaluate(serverCtx, &req)
		}

		validationDuration := time.Since(start)

		if evalErr != nil {
			serverCtx.Logger.Error("AuthZEN evaluation error",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("subject_id", req.Subject.ID),
				logging.F("error", evalErr.Error()))

			// Record error metrics
			if serverCtx.Metrics != nil {
				serverCtx.Metrics.RecordError("evaluation_error", "authzen_decision")
			}

			c.JSON(500, buildResponse(false, evalErr.Error()))
			return
		}

		if resp.Decision {
			serverCtx.Logger.Info("AuthZEN request approved",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("subject_id", req.Subject.ID),
				logging.F("resource_type", req.Resource.Type),
				logging.F("duration_ms", validationDuration.Milliseconds()))

			// Record successful validation metrics
			if serverCtx.Metrics != nil {
				serverCtx.Metrics.RecordCertValidation(validationDuration, true)
			}
		} else {
			serverCtx.Logger.Info("AuthZEN request denied",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("subject_id", req.Subject.ID),
				logging.F("resource_type", req.Resource.Type),
				logging.F("duration_ms", validationDuration.Milliseconds()))

			// Record failed validation metrics
			if serverCtx.Metrics != nil {
				serverCtx.Metrics.RecordCertValidation(validationDuration, false)
			}
		}

		c.JSON(200, resp)
	}
}

// legacyEvaluate implements the old direct CertPool validation for backward compatibility
func legacyEvaluate(serverCtx *ServerContext, req *authzen.EvaluationRequest) (*authzen.EvaluationResponse, error) {
	// Validate request against AuthZEN Trust Registry Profile
	if err := req.Validate(); err != nil {
		return &authzen.EvaluationResponse{
			Decision: false,
			Context: &authzen.EvaluationResponseContext{
				Reason: map[string]interface{}{
					"error": fmt.Sprintf("validation error: %v", err),
				},
			},
		}, nil
	}

	// Extract certificates from resource.key based on resource.type
	var certs []*x509.Certificate
	var parseErr error

	if req.Resource.Type == "x5c" {
		// resource.key is an array of base64-encoded X.509 certificates
		certs, parseErr = x509util.ParseX5CFromArray(req.Resource.Key)
	} else {
		// resource.type == "jwk" - extract certificate from JWK x5c claim
		certs, parseErr = x509util.ParseX5CFromJWK(req.Resource.Key)
	}

	if parseErr != nil {
		return &authzen.EvaluationResponse{
			Decision: false,
			Context: &authzen.EvaluationResponseContext{
				Reason: map[string]interface{}{
					"error": parseErr.Error(),
				},
			},
		}, nil
	}

	if len(certs) == 0 {
		return &authzen.EvaluationResponse{
			Decision: false,
			Context: &authzen.EvaluationResponseContext{
				Reason: map[string]interface{}{
					"error": "no certificates found in resource.key",
				},
			},
		}, nil
	}

	// Validate certificate chain against TSL certificate pool
	serverCtx.RLock()
	certPool := serverCtx.PipelineContext.CertPool
	serverCtx.RUnlock()

	if certPool == nil {
		return &authzen.EvaluationResponse{
			Decision: false,
			Context: &authzen.EvaluationResponseContext{
				Reason: map[string]interface{}{
					"error": "CertPool is nil",
				},
			},
		}, nil
	}

	opts := x509.VerifyOptions{
		Roots: certPool,
	}
	_, err := certs[0].Verify(opts)

	if err == nil {
		resp := buildResponse(true, "")
		return &resp, nil
	} else {
		resp := buildResponse(false, err.Error())
		return &resp, nil
	}
}

// InfoHandler godoc
// @Summary Get TSL information (DEPRECATED - use GET /tsls)
// @Description Returns detailed summaries of all loaded Trust Status Lists
// @Description
// @Description DEPRECATED: This endpoint is deprecated. Use GET /tsls instead.
// @Description
// @Description This endpoint provides comprehensive information about each TSL including:
// @Description - Territory code
// @Description - Sequence number
// @Description - Issue date
// @Description - Next update date
// @Description - Number of services
// @Tags Status
// @Deprecated true
// @Produce json
// @Success 200 {object} map[string]interface{} "tsl_summaries"
// @Router /info [get]
func InfoHandler(serverCtx *ServerContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add deprecation headers
		c.Header("Deprecation", "true")
		c.Header("Link", "</tsls>; rel=\"alternate\"")
		c.Header("X-API-Warn", "This endpoint is deprecated. Please use GET /tsls instead.")

		serverCtx.RLock()
		defer serverCtx.RUnlock()
		summaries := make([]map[string]interface{}, 0)

		// Add debug logging to inspect the pipeline context
		tslSize := 0
		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			tslSize = serverCtx.PipelineContext.TSLs.Size()
		}

		serverCtx.Logger.Debug("API info request (deprecated): Inspecting pipeline context",
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
		serverCtx.Logger.Warn("API info request (deprecated endpoint)",
			logging.F("remote_ip", c.ClientIP()),
			logging.F("summary_count", len(summaries)),
			logging.F("replacement", "GET /tsls"))

		c.JSON(200, gin.H{
			"tsl_summaries": summaries,
		})
	}
}

// TSLsHandler godoc
// @Summary List Trust Status Lists
// @Description Returns comprehensive information about all loaded Trust Status Lists
// @Description
// @Description This is the primary endpoint for retrieving TSL metadata including:
// @Description - Territory codes
// @Description - Sequence numbers
// @Description - Issue and next update dates
// @Description - Service counts per TSL
// @Description - Last processing timestamp
// @Tags TSLs
// @Produce json
// @Success 200 {object} map[string]interface{} "count, last_updated, tsls"
// @Router /tsls [get]
func TSLsHandler(serverCtx *ServerContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverCtx.RLock()
		defer serverCtx.RUnlock()

		summaries := make([]map[string]interface{}, 0)
		tslCount := 0
		lastUpdated := serverCtx.LastProcessed.Format(time.RFC3339)

		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			tslCount = serverCtx.PipelineContext.TSLs.Size()
			for _, tsl := range serverCtx.PipelineContext.TSLs.ToSlice() {
				if tsl != nil {
					summaries = append(summaries, tsl.Summary())
				}
			}
		}

		serverCtx.Logger.Info("API /tsls request",
			logging.F("remote_ip", c.ClientIP()),
			logging.F("tsl_count", tslCount))

		c.JSON(200, gin.H{
			"count":        tslCount,
			"last_updated": lastUpdated,
			"tsls":         summaries,
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
