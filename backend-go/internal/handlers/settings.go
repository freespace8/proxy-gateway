// Package handlers 提供 HTTP 处理器
package handlers

import (
	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/gin-gonic/gin"
)

func mappingOrEmpty(mapping map[string]string) map[string]string {
	if len(mapping) == 0 {
		return map[string]string{}
	}
	return mapping
}

// GetFuzzyMode 获取 Fuzzy 模式状态
func GetFuzzyMode(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"fuzzyModeEnabled": cfgManager.GetFuzzyModeEnabled(),
		})
	}
}

// SetFuzzyMode 设置 Fuzzy 模式状态
func SetFuzzyMode(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		if err := cfgManager.SetFuzzyModeEnabled(req.Enabled); err != nil {
			c.JSON(500, gin.H{"error": "Failed to save config"})
			return
		}

		c.JSON(200, gin.H{
			"success":          true,
			"fuzzyModeEnabled": req.Enabled,
		})
	}
}

// GetGlobalModelMapping 获取全局模型重定向
func GetGlobalModelMapping(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"globalModelMapping": mappingOrEmpty(cfgManager.GetGlobalModelMapping()),
		})
	}
}

// SetGlobalModelMapping 设置全局模型重定向
func SetGlobalModelMapping(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			GlobalModelMapping map[string]string `json:"globalModelMapping"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		if err := cfgManager.SetGlobalModelMapping(req.GlobalModelMapping); err != nil {
			c.JSON(500, gin.H{"error": "Failed to save config"})
			return
		}

		c.JSON(200, gin.H{
			"success":            true,
			"globalModelMapping": mappingOrEmpty(cfgManager.GetGlobalModelMapping()),
		})
	}
}

// GetGlobalReasoningMapping 获取全局思考重定向
func GetGlobalReasoningMapping(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"globalReasoningMapping": mappingOrEmpty(cfgManager.GetGlobalReasoningMapping()),
		})
	}
}

// SetGlobalReasoningMapping 设置全局思考重定向
func SetGlobalReasoningMapping(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			GlobalReasoningMapping map[string]string `json:"globalReasoningMapping"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		if err := cfgManager.SetGlobalReasoningMapping(req.GlobalReasoningMapping); err != nil {
			c.JSON(500, gin.H{"error": "Failed to save config"})
			return
		}

		c.JSON(200, gin.H{
			"success":                true,
			"globalReasoningMapping": mappingOrEmpty(cfgManager.GetGlobalReasoningMapping()),
		})
	}
}
