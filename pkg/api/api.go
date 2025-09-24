package api

import (
	"github.com/gin-gonic/gin"
)

// RegisterAPIRoutes registers all API endpoints on the given Gin router using ServerContext.
func RegisterAPIRoutes(r *gin.Engine, serverCtx *ServerContext) {
	r.GET("/status", func(c *gin.Context) {
		serverCtx.RLock()
		defer serverCtx.RUnlock()
		tslCount := 0
		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			tslCount = len(serverCtx.PipelineContext.TSLs)
		}
		c.JSON(200, gin.H{
			"tsl_count":      tslCount,
			"last_processed": serverCtx.LastProcessed.Format("2006-01-02T15:04:05Z07:00"),
		})
	})

	r.POST("/authzen/decision", func(c *gin.Context) {
		var req map[string]interface{}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}
		c.JSON(200, gin.H{
			"decision": "Permit",
			"reason":   "minimal implementation",
		})
	})

	r.GET("/info", func(c *gin.Context) {
		serverCtx.RLock()
		defer serverCtx.RUnlock()
		summaries := make([]map[string]interface{}, 0)
		if serverCtx.PipelineContext != nil && serverCtx.PipelineContext.TSLs != nil {
			for _, tsl := range serverCtx.PipelineContext.TSLs {
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
