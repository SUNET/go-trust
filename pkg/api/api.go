package api

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/SUNET/go-trust/pkg/authzen"
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

// StartBackgroundUpdater starts a goroutine that periodically updates the ServerContext using the pipeline.
// It executes the pipeline at the specified frequency, updating the ServerContext with the new PipelineContext.
// If the pipeline execution fails, an error message is printed to stderr, but the goroutine continues running.
//
// Parameters:
//   - pl: The pipeline to process periodically
//   - serverCtx: The server context to update with pipeline results
//   - freq: The frequency at which to process the pipeline (e.g., 5m for every 5 minutes)
//
// This function is typically called at server startup to ensure TSLs are kept up-to-date.
func StartBackgroundUpdater(pl *pipeline.Pipeline, serverCtx *ServerContext, freq time.Duration) error {
	go func() {
		for {
			newCtx, err := pl.Process(&pipeline.Context{})
			serverCtx.Lock()
			if err == nil && newCtx != nil {
				serverCtx.PipelineContext = newCtx
				serverCtx.LastProcessed = time.Now()
			}
			serverCtx.Unlock()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Pipeline error: %v\n", err)
			}
			time.Sleep(freq)
		}
	}()
	return nil
}

// RegisterAPIRoutes registers all API endpoints on the given Gin router using ServerContext.
// It sets up the following endpoints:
//
// GET /status - Returns the current server status including TSL count and last processing time
//
// GET /info - Returns detailed summaries of all TSLs in the current pipeline context
//
// POST /authzen/decision - Implements the AuthZEN protocol for making trust decisions
//   This endpoint processes AuthZEN EvaluationRequest objects containing x5c certificate
//   chains and verifies them against the trusted certificates in the pipeline context.
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
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}

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
					c.JSON(200, buildResponse(true, ""))
				} else {
					c.JSON(200, buildResponse(false, err.Error()))
				}
				return
			} else {
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
		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			for _, tsl := range serverCtx.PipelineContext.TSLs.ToSlice() {
				if tsl != nil {
					summaries = append(summaries, tsl.Summary())
				}
			}
		}
		c.JSON(200, gin.H{
			"tsl_summaries": summaries,
		})
	})
}
