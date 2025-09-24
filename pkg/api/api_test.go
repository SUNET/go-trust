package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"github.com/SUNET/go-trust/pkg/pipeline"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestServer() (*gin.Engine, *ServerContext) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	serverCtx := &ServerContext{
		PipelineContext: &pipeline.Context{},
		LastProcessed:   time.Now(),
	}
	RegisterAPIRoutes(r, serverCtx)
	return r, serverCtx
}

func TestStatusEndpoint(t *testing.T) {
	r, serverCtx := setupTestServer()
	serverCtx.Lock()
	serverCtx.PipelineContext.TSLs = make([]*etsi119612.TSL, 2) // Simulate 2 TSLs loaded
	serverCtx.Unlock()

	req, _ := http.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "tsl_count")
	assert.Contains(t, w.Body.String(), "last_processed")
}

func TestInfoEndpoint_Empty(t *testing.T) {
	r, _ := setupTestServer()
	req, _ := http.NewRequest("GET", "/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "tsl_summaries")
}

func TestAuthzenDecisionEndpoint(t *testing.T) {
	r, _ := setupTestServer()
	body := `{"subject":"alice","action":"read"}`
	req, _ := http.NewRequest("POST", "/authzen/decision", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "Permit")
}
