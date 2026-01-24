package handlers

import (
	"net/http"
	"strconv"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/metrics"
	"github.com/BenedictKing/claude-proxy/internal/scheduler"
	"github.com/gin-gonic/gin"
)

func GetKeyCircuitLog(store metrics.KeyCircuitLogStore, cfgManager *config.ConfigManager, apiType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "key circuit log disabled"})
			return
		}

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
		upstream := upstreams[channelIndex]
		if keyIndex < 0 || keyIndex >= len(upstream.APIKeys) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
			return
		}

		keyID := metrics.HashAPIKey(upstream.APIKeys[keyIndex])
		logStr, ok, err := store.GetKeyCircuitLog(apiType, keyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query last failure log"})
			return
		}
		if !ok || logStr == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "No failure log"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"log": logStr})
	}
}

func ResetKeyCircuitState(sch *scheduler.ChannelScheduler, cfgManager *config.ConfigManager, apiType string, requestLogStats metrics.RequestLogStatsProvider) gin.HandlerFunc {
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
		upstream := upstreams[channelIndex]
		if keyIndex < 0 || keyIndex >= len(upstream.APIKeys) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
			return
		}

		apiKey := upstream.APIKeys[keyIndex]
		baseURLs := upstream.GetAllBaseURLs()
		switch apiType {
		case "messages":
			for _, baseURL := range baseURLs {
				sch.ResetKeyMetrics(baseURL, apiKey, false)
			}
		case "responses":
			for _, baseURL := range baseURLs {
				sch.ResetKeyMetrics(baseURL, apiKey, true)
			}
		case "gemini":
			for _, baseURL := range baseURLs {
				sch.GetGeminiMetricsManager().ResetKey(baseURL, apiKey)
			}
		}

		cfgManager.ClearFailedKey(apiKey)

		// 清空按“请求日志”口径统计的累计请求数，并隐藏历史日志（本次进程内）。
		if requestLogStats != nil && apiKey != "" {
			requestLogStats.ResetKey(apiType, channelIndex, metrics.HashAPIKey(apiKey))
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "key 统计与状态已重置",
		})
	}
}

func ResetKeyCircuitStatus(sch *scheduler.ChannelScheduler, cfgManager *config.ConfigManager, apiType string) gin.HandlerFunc {
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
		upstream := upstreams[channelIndex]
		if keyIndex < 0 || keyIndex >= len(upstream.APIKeys) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
			return
		}

		apiKey := upstream.APIKeys[keyIndex]
		baseURLs := upstream.GetAllBaseURLs()
		switch apiType {
		case "messages":
			for _, baseURL := range baseURLs {
				sch.ResetKeyState(baseURL, apiKey, false)
			}
		case "responses":
			for _, baseURL := range baseURLs {
				sch.ResetKeyState(baseURL, apiKey, true)
			}
		case "gemini":
			for _, baseURL := range baseURLs {
				sch.GetGeminiMetricsManager().ResetKeyState(baseURL, apiKey)
			}
		}

		cfgManager.ClearFailedKey(apiKey)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "key 状态已重置",
		})
	}
}

func ResetAllKeysCircuitStatus(sch *scheduler.ChannelScheduler, cfgManager *config.ConfigManager, apiType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelIndex, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
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

		upstream := upstreams[channelIndex]
		baseURLs := upstream.GetAllBaseURLs()
		resetCount := 0

		for _, apiKey := range upstream.APIKeys {
			if apiKey == "" {
				continue
			}
			switch apiType {
			case "messages":
				for _, baseURL := range baseURLs {
					sch.ResetKeyState(baseURL, apiKey, false)
				}
			case "responses":
				for _, baseURL := range baseURLs {
					sch.ResetKeyState(baseURL, apiKey, true)
				}
			case "gemini":
				for _, baseURL := range baseURLs {
					sch.GetGeminiMetricsManager().ResetKeyState(baseURL, apiKey)
				}
			}

			cfgManager.ClearFailedKey(apiKey)
			resetCount++
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   resetCount,
			"message": "渠道下所有 key 状态已重置",
		})
	}
}

func ResetAllKeysCircuitState(sch *scheduler.ChannelScheduler, cfgManager *config.ConfigManager, apiType string, requestLogStats metrics.RequestLogStatsProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelIndex, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
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

		upstream := upstreams[channelIndex]
		baseURLs := upstream.GetAllBaseURLs()
		resetCount := 0

		for _, apiKey := range upstream.APIKeys {
			if apiKey == "" {
				continue
			}
			switch apiType {
			case "messages":
				for _, baseURL := range baseURLs {
					sch.ResetKeyMetrics(baseURL, apiKey, false)
				}
			case "responses":
				for _, baseURL := range baseURLs {
					sch.ResetKeyMetrics(baseURL, apiKey, true)
				}
			case "gemini":
				for _, baseURL := range baseURLs {
					sch.GetGeminiMetricsManager().ResetKey(baseURL, apiKey)
				}
			}

			cfgManager.ClearFailedKey(apiKey)
			resetCount++
		}

		// 清空该渠道下按“请求日志”口径统计的累计请求数，并隐藏历史日志（本次进程内）。
		if requestLogStats != nil {
			requestLogStats.ResetChannel(apiType, channelIndex)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"count":   resetCount,
			"message": "渠道下所有 key 统计与状态已重置",
		})
	}
}
