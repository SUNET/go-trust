package api

import (
	"time"

	"github.com/SUNET/go-trust/pkg/logging"
	"github.com/gin-gonic/gin"
)

// HealthResponse represents the response from health check endpoints
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ReadinessResponse represents the response from the readiness endpoint
type ReadinessResponse struct {
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	TSLCount      int       `json:"tsl_count"`
	LastProcessed string    `json:"last_processed,omitempty"`
	Ready         bool      `json:"ready"`
	Message       string    `json:"message,omitempty"`
}

// RegisterHealthEndpoints registers health check endpoints on the given Gin router.
// These endpoints are useful for Kubernetes liveness and readiness probes, load balancers,
// and monitoring systems.
//
// Endpoints:
//
//	GET /health       - Liveness probe: returns 200 if the server is running
//	GET /healthz      - Alias for /health
//	GET /ready        - Readiness probe: returns 200 if server is ready to accept traffic
//	GET /readiness    - Alias for /ready
//
// The /health endpoint always returns 200 OK if the server is running, indicating
// that the process is alive and can handle requests.
//
// The /ready endpoint checks whether the service has:
//   - Successfully loaded at least one TSL
//   - Processed the pipeline at least once
//
// If these conditions are not met, it returns 503 Service Unavailable.
func RegisterHealthEndpoints(r *gin.Engine, serverCtx *ServerContext) {
	r.GET("/health", HealthHandler(serverCtx))
	r.GET("/healthz", HealthHandler(serverCtx))
	r.GET("/ready", ReadinessHandler(serverCtx))
	r.GET("/readiness", ReadinessHandler(serverCtx))

	serverCtx.Logger.Info("Health check endpoints registered",
		logging.F("endpoints", []string{"/health", "/healthz", "/ready", "/readiness"}))
}

// HealthHandler godoc
// @Summary Liveness check
// @Description Returns OK if the server is running and able to handle requests
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
// @Router /healthz [get]
func HealthHandler(serverCtx *ServerContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverCtx.Logger.Debug("Health check requested",
			logging.F("remote_ip", c.ClientIP()),
			logging.F("endpoint", c.Request.URL.Path))

		c.JSON(200, HealthResponse{
			Status:    "ok",
			Timestamp: time.Now(),
		})
	}
}

// ReadinessHandler godoc
// @Summary Readiness check
// @Description Returns ready status if pipeline has been processed and TSLs are loaded
// @Tags Health
// @Produce json
// @Success 200 {object} ReadinessResponse "Service is ready"
// @Failure 503 {object} ReadinessResponse "Service is not ready"
// @Router /ready [get]
// @Router /readiness [get]
func ReadinessHandler(serverCtx *ServerContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverCtx.RLock()
		tslCount := 0
		lastProcessed := ""
		pipelineProcessed := !serverCtx.LastProcessed.IsZero()

		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			tslCount = serverCtx.PipelineContext.TSLs.Size()
		}

		if pipelineProcessed {
			lastProcessed = serverCtx.LastProcessed.Format(time.RFC3339)
		}
		serverCtx.RUnlock()

		// Service is ready if:
		// 1. Pipeline has been processed at least once
		// 2. At least one TSL is loaded (optional but recommended)
		isReady := pipelineProcessed && tslCount > 0

		response := ReadinessResponse{
			Timestamp:     time.Now(),
			TSLCount:      tslCount,
			LastProcessed: lastProcessed,
			Ready:         isReady,
		}

		if isReady {
			response.Status = "ready"
			response.Message = "Service is ready to accept traffic"

			serverCtx.Logger.Debug("Readiness check passed",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("endpoint", c.Request.URL.Path),
				logging.F("tsl_count", tslCount),
				logging.F("last_processed", lastProcessed))

			c.JSON(200, response)
		} else {
			response.Status = "not_ready"
			if !pipelineProcessed {
				response.Message = "Pipeline has not been processed yet"
			} else if tslCount == 0 {
				response.Message = "No TSLs loaded yet"
			}

			serverCtx.Logger.Warn("Readiness check failed",
				logging.F("remote_ip", c.ClientIP()),
				logging.F("endpoint", c.Request.URL.Path),
				logging.F("reason", response.Message),
				logging.F("tsl_count", tslCount),
				logging.F("pipeline_processed", pipelineProcessed))

			c.JSON(503, response)
		}
	}
}
