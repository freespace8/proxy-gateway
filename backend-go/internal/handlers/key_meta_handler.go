package handlers

import (
	"net/http"
	"strconv"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/gin-gonic/gin"
)

func PatchAPIKeyMeta(cfgManager *config.ConfigManager, apiType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelIndex, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
			return
		}
		keyIndex, err := strconv.Atoi(c.Param("keyIndex"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key index"})
			return
		}

		var req struct {
			Disabled *bool `json:"disabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Disabled == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		cfg := cfgManager.GetConfig()
		var upstreams []config.UpstreamConfig
		switch apiType {
		case "messages":
			upstreams = cfg.Upstream
		case "responses":
			upstreams = cfg.ResponsesUpstream
		case "gemini":
			upstreams = cfg.GeminiUpstream
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid api type"})
			return
		}

		if channelIndex < 0 || channelIndex >= len(upstreams) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		if keyIndex < 0 || keyIndex >= len(upstreams[channelIndex].APIKeys) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
			return
		}

		if err := cfgManager.SetAPIKeyDisabled(apiType, channelIndex, keyIndex, *req.Disabled); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
