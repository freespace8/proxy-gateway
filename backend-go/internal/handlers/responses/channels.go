// Package responses 提供 Responses API 的渠道管理
package responses

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/httpclient"
	"github.com/BenedictKing/claude-proxy/internal/scheduler"
	"github.com/gin-gonic/gin"
)

// GetUpstreams 获取 Responses 上游列表
func GetUpstreams(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := cfgManager.GetConfig()

		upstreams := make([]gin.H, len(cfg.ResponsesUpstream))
		for i, up := range cfg.ResponsesUpstream {
			status := config.GetChannelStatus(&up)
			priority := config.GetChannelPriority(&up, i)

			upstreams[i] = gin.H{
				"index":              i,
				"name":               up.Name,
				"serviceType":        up.ServiceType,
				"baseUrl":            up.BaseURL,
				"baseUrls":           up.BaseURLs,
				"apiKeys":            up.APIKeys,
				"description":        up.Description,
				"website":            up.Website,
				"insecureSkipVerify": up.InsecureSkipVerify,
				"modelMapping":       up.ModelMapping,
				"latency":            nil,
				"status":             status,
				"priority":           priority,
				"promotionUntil":     up.PromotionUntil,
				"lowQuality":         up.LowQuality,
			}
		}

		c.JSON(200, gin.H{
			"channels":    upstreams,
			"loadBalance": cfg.ResponsesLoadBalance,
		})
	}
}

// AddUpstream 添加 Responses 上游
func AddUpstream(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var upstream config.UpstreamConfig
		if err := c.ShouldBindJSON(&upstream); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		if err := cfgManager.AddResponsesUpstream(upstream); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Responses upstream added successfully"})
	}
}

// UpdateUpstream 更新 Responses 上游
func UpdateUpstream(cfgManager *config.ConfigManager, sch *scheduler.ChannelScheduler) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid upstream ID"})
			return
		}

		var updates config.UpstreamUpdate
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		shouldResetMetrics, err := cfgManager.UpdateResponsesUpstream(id, updates)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// 单 key 更换时重置熔断状态
		if shouldResetMetrics {
			sch.ResetChannelMetrics(id, true)
		}

		c.JSON(200, gin.H{"message": "Responses upstream updated successfully"})
	}
}

// DeleteUpstream 删除 Responses 上游
func DeleteUpstream(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid upstream ID"})
			return
		}

		if _, err := cfgManager.RemoveResponsesUpstream(id); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Responses upstream deleted successfully"})
	}
}

// AddApiKey 添加 Responses 渠道 API 密钥
func AddApiKey(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid upstream ID"})
			return
		}

		var req struct {
			APIKey string `json:"apiKey"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		if err := cfgManager.AddResponsesAPIKey(id, req.APIKey); err != nil {
			if strings.Contains(err.Error(), "无效的上游索引") {
				c.JSON(404, gin.H{"error": "Upstream not found"})
			} else if strings.Contains(err.Error(), "API密钥已存在") {
				c.JSON(400, gin.H{"error": "API密钥已存在"})
			} else {
				c.JSON(500, gin.H{"error": "Failed to save config"})
			}
			return
		}

		c.JSON(200, gin.H{
			"message": "API密钥已添加",
			"success": true,
		})
	}
}

// DeleteApiKey 删除 Responses 渠道 API 密钥
func DeleteApiKey(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid upstream ID"})
			return
		}

		apiKey := c.Param("apiKey")
		if apiKey == "" {
			c.JSON(400, gin.H{"error": "API key is required"})
			return
		}

		if err := cfgManager.RemoveResponsesAPIKey(id, apiKey); err != nil {
			if strings.Contains(err.Error(), "无效的上游索引") {
				c.JSON(404, gin.H{"error": "Upstream not found"})
			} else if strings.Contains(err.Error(), "API密钥不存在") {
				c.JSON(404, gin.H{"error": "API key not found"})
			} else {
				c.JSON(500, gin.H{"error": "Failed to save config"})
			}
			return
		}

		c.JSON(200, gin.H{
			"message": "API密钥已删除",
		})
	}
}

// MoveApiKeyToTop 将 Responses 渠道 API 密钥移到最前面
func MoveApiKeyToTop(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		apiKey := c.Param("apiKey")

		if err := cfgManager.MoveResponsesAPIKeyToTop(id, apiKey); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "API密钥已置顶"})
	}
}

// MoveApiKeyToBottom 将 Responses 渠道 API 密钥移到最后面
func MoveApiKeyToBottom(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		apiKey := c.Param("apiKey")

		if err := cfgManager.MoveResponsesAPIKeyToBottom(id, apiKey); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "API密钥已置底"})
	}
}

// UpdateLoadBalance 更新 Responses 负载均衡策略
func UpdateLoadBalance(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Strategy string `json:"strategy"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		if err := cfgManager.SetResponsesLoadBalance(req.Strategy); err != nil {
			if strings.Contains(err.Error(), "无效的负载均衡策略") {
				c.JSON(400, gin.H{"error": err.Error()})
			} else {
				c.JSON(500, gin.H{"error": "Failed to save config"})
			}
			return
		}

		c.JSON(200, gin.H{
			"message":  "Responses 负载均衡策略已更新",
			"strategy": req.Strategy,
		})
	}
}

// ReorderChannels 重新排序 Responses 渠道优先级
func ReorderChannels(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Order []int `json:"order"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		if err := cfgManager.ReorderResponsesUpstreams(req.Order); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Responses 渠道优先级已更新",
		})
	}
}

// SetChannelStatus 设置 Responses 渠道状态
func SetChannelStatus(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid channel ID"})
			return
		}

		var req struct {
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		if err := cfgManager.SetResponsesChannelStatus(id, req.Status); err != nil {
			if strings.Contains(err.Error(), "无效的上游索引") {
				c.JSON(404, gin.H{"error": "Channel not found"})
			} else {
				c.JSON(400, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Responses 渠道状态已更新",
			"status":  req.Status,
		})
	}
}

// PingChannel Ping单个 Responses 渠道
func PingChannel(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
			return
		}

		cfg := cfgManager.GetConfig()
		if id < 0 || id >= len(cfg.ResponsesUpstream) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}

		channel := cfg.ResponsesUpstream[id]
		result := pingChannelURLs(&channel)
		c.JSON(http.StatusOK, result)
	}
}

// pingChannelURLs 测试渠道的所有 BaseURL，返回最快的延迟
func pingChannelURLs(ch *config.UpstreamConfig) gin.H {
	urls := ch.GetAllBaseURLs()
	if len(urls) == 0 {
		return gin.H{"success": false, "latency": 0, "status": "error", "error": "no_base_url"}
	}

	if len(urls) == 1 {
		return pingURL(urls[0], ch.InsecureSkipVerify)
	}

	type pingResult struct {
		url     string
		latency int64
		success bool
		err     string
	}

	results := make(chan pingResult, len(urls))
	for _, url := range urls {
		go func(testURL string) {
			startTime := time.Now()
			testURL = strings.TrimSuffix(testURL, "/")
			client := httpclient.GetManager().GetStandardClient(5*time.Second, ch.InsecureSkipVerify)
			req, err := http.NewRequest("HEAD", testURL, nil)
			if err != nil {
				results <- pingResult{url: testURL, latency: 0, success: false, err: "req_creation_failed"}
				return
			}
			resp, err := client.Do(req)
			latency := time.Since(startTime).Milliseconds()
			if err != nil {
				results <- pingResult{url: testURL, latency: latency, success: false, err: err.Error()}
				return
			}
			resp.Body.Close()
			results <- pingResult{url: testURL, latency: latency, success: true}
		}(url)
	}

	var bestResult *pingResult
	for i := 0; i < len(urls); i++ {
		r := <-results
		if r.success {
			if bestResult == nil || !bestResult.success || r.latency < bestResult.latency {
				bestResult = &r
			}
		} else if bestResult == nil || !bestResult.success {
			bestResult = &r
		}
	}

	if bestResult == nil {
		return gin.H{"success": false, "latency": 0, "status": "error", "error": "all_urls_failed"}
	}

	if bestResult.success {
		return gin.H{"success": true, "latency": bestResult.latency, "status": "healthy"}
	}
	return gin.H{"success": false, "latency": bestResult.latency, "status": "error", "error": bestResult.err}
}

// pingURL 测试单个 URL
func pingURL(testURL string, insecureSkipVerify bool) gin.H {
	startTime := time.Now()
	testURL = strings.TrimSuffix(testURL, "/")
	client := httpclient.GetManager().GetStandardClient(5*time.Second, insecureSkipVerify)
	req, err := http.NewRequest("HEAD", testURL, nil)
	if err != nil {
		return gin.H{"success": false, "latency": 0, "status": "error", "error": "req_creation_failed"}
	}
	resp, err := client.Do(req)
	latency := time.Since(startTime).Milliseconds()
	if err != nil {
		return gin.H{"success": false, "latency": latency, "status": "error", "error": err.Error()}
	}
	resp.Body.Close()
	return gin.H{"success": true, "latency": latency, "status": "healthy"}
}

// PingAllChannels Ping所有 Responses 渠道
func PingAllChannels(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := cfgManager.GetConfig()
		results := make(chan gin.H)
		var wg sync.WaitGroup

		for i, channel := range cfg.ResponsesUpstream {
			wg.Add(1)
			go func(id int, ch config.UpstreamConfig) {
				defer wg.Done()
				result := pingChannelURLs(&ch)
				result["id"] = id
				result["name"] = ch.Name
				results <- result
			}(i, channel)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		var finalResults []gin.H
		for res := range results {
			finalResults = append(finalResults, res)
		}

		c.JSON(http.StatusOK, finalResults)
	}
}
