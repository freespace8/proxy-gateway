package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BenedictKing/claude-proxy/internal/billing"
	"github.com/BenedictKing/claude-proxy/internal/cache"
	"github.com/BenedictKing/claude-proxy/internal/config"
	"github.com/BenedictKing/claude-proxy/internal/handlers"
	"github.com/BenedictKing/claude-proxy/internal/handlers/gemini"
	"github.com/BenedictKing/claude-proxy/internal/handlers/messages"
	"github.com/BenedictKing/claude-proxy/internal/handlers/responses"
	"github.com/BenedictKing/claude-proxy/internal/logger"
	"github.com/BenedictKing/claude-proxy/internal/metrics"
	"github.com/BenedictKing/claude-proxy/internal/middleware"
	"github.com/BenedictKing/claude-proxy/internal/monitor"
	"github.com/BenedictKing/claude-proxy/internal/pricing"
	"github.com/BenedictKing/claude-proxy/internal/scheduler"
	"github.com/BenedictKing/claude-proxy/internal/session"
	"github.com/BenedictKing/claude-proxy/internal/usage"
	"github.com/BenedictKing/claude-proxy/internal/warmup"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("没有找到 .env 文件，使用环境变量或默认值")
	}

	// 设置版本信息到 handlers 包
	handlers.SetVersionInfo(Version, BuildTime, GitCommit)

	// 初始化配置管理器
	envCfg := config.NewEnvConfig()

	// 初始化日志系统（必须在其他初始化之前）
	logCfg := &logger.Config{
		LogDir:     envCfg.LogDir,
		LogFile:    envCfg.LogFile,
		MaxSize:    envCfg.LogMaxSize,
		MaxBackups: envCfg.LogMaxBackups,
		MaxAge:     envCfg.LogMaxAge,
		Compress:   envCfg.LogCompress,
		Console:    envCfg.LogToConsole,
	}
	if err := logger.Setup(logCfg); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}

	cfgManager, err := config.NewConfigManager(".config/config.json")
	if err != nil {
		log.Fatalf("初始化配置管理器失败: %v", err)
	}
	defer cfgManager.Close()

	// 初始化会话管理器（Responses API 专用）
	sessionManager := session.NewSessionManager(
		24*time.Hour, // 24小时过期
		100,          // 最多100条消息
		100000,       // 最多100k tokens
	)
	log.Printf("[Session-Init] 会话管理器已初始化")

	// 指标与 Key 熔断日志：纯内存保留（默认 7 天；由 METRICS_RETENTION_DAYS 控制）
	metricsRetention := time.Duration(envCfg.MetricsRetentionDays) * 24 * time.Hour
	keyCircuitLogStore := metrics.NewMemoryKeyCircuitLogStore(metricsRetention)

	// 初始化多渠道调度器（Messages、Responses、Gemini 使用独立的指标管理器）
	var messagesMetricsManager, responsesMetricsManager, geminiMetricsManager *metrics.MetricsManager
	messagesMetricsManager = metrics.NewMetricsManagerWithConfig(envCfg.MetricsWindowSize, envCfg.MetricsFailureThreshold)
	responsesMetricsManager = metrics.NewMetricsManagerWithConfig(envCfg.MetricsWindowSize, envCfg.MetricsFailureThreshold)
	geminiMetricsManager = metrics.NewMetricsManagerWithConfig(envCfg.MetricsWindowSize, envCfg.MetricsFailureThreshold)
	messagesMetricsManager.SetRetentionDays(envCfg.MetricsRetentionDays)
	responsesMetricsManager.SetRetentionDays(envCfg.MetricsRetentionDays)
	geminiMetricsManager.SetRetentionDays(envCfg.MetricsRetentionDays)
	traceAffinityManager := session.NewTraceAffinityManager()

	// 初始化 URL 管理器（非阻塞，动态排序）
	urlManager := warmup.NewURLManager(30*time.Second, 3) // 30秒冷却期，连续3次失败后移到末尾
	log.Printf("[URLManager-Init] URL管理器已初始化 (冷却期: 30秒, 最大连续失败: 3)")

	channelScheduler := scheduler.NewChannelScheduler(cfgManager, messagesMetricsManager, responsesMetricsManager, geminiMetricsManager, traceAffinityManager, urlManager)
	log.Printf("[Scheduler-Init] 多渠道调度器已初始化 (失败率阈值: %.0f%%, 滑动窗口: %d)",
		messagesMetricsManager.GetFailureThreshold()*100, messagesMetricsManager.GetWindowSize())

	// 初始化 /v1/models 响应缓存（模型列表变化频率低，使用较长 TTL）
	modelsCacheMetrics := &metrics.CacheMetrics{}
	modelsResponseCache := cache.NewHTTPResponseCache(200, 10*time.Minute, modelsCacheMetrics)

	// 实时请求监控
	liveRequestManager := monitor.NewLiveRequestManager(50)

	// 请求日志存储：固定使用内存环形缓冲（最近 N 条）。
	requestLogStore := metrics.NewMemoryRequestLogStore(envCfg.RequestLogsMemoryMaxSize)

	// 初始化计费相关组件
	var billingClient *billing.Client
	var usageStore *usage.Store

	// 价格表服务始终初始化（用于成本统计，即使不启用计费）
	pricingInterval, err := time.ParseDuration(envCfg.PricingUpdateInterval)
	if err != nil {
		pricingInterval = 24 * time.Hour
	}
	pricingService := pricing.NewService(pricingInterval)
	log.Printf("[Pricing-Init] 价格表服务已初始化 (更新间隔: %s)", pricingInterval)

	if envCfg.IsBillingEnabled() {
		billingClient = billing.NewClient(envCfg.SweAgentBillingURL)
		log.Printf("[Billing-Init] 计费客户端已初始化: %s", envCfg.SweAgentBillingURL)

		usageStore = usage.NewStore(10000)
		log.Printf("[Usage-Init] 使用量存储已初始化")
	}

	// billingHandler 始终创建（用于成本计算），但 client/usageStore 可能为 nil
	billingHandler := billing.NewHandler(billingClient, pricingService, usageStore, envCfg.PreAuthAmountCents)
	if envCfg.IsBillingEnabled() {
		log.Printf("[Billing-Init] 计费处理器已初始化 (预授权: %d cents)", envCfg.PreAuthAmountCents)
	}

	// 设置 Gin 模式
	if envCfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器（使用自定义 Logger，根据 QUIET_POLLING_LOGS 配置过滤轮询日志）
	r := gin.New()
	r.Use(middleware.FilteredLogger(envCfg))
	r.Use(gin.Recovery())

	// 配置 CORS
	r.Use(middleware.CORSMiddleware(envCfg))

	// Web UI 访问控制中间件
	r.Use(middleware.WebAuthMiddleware(envCfg, cfgManager))

	// 健康检查端点（固定路径 /health，与 Dockerfile HEALTHCHECK 保持一致）
	r.GET("/health", handlers.HealthCheck(envCfg, cfgManager))

	// 开发信息端点
	if envCfg.IsDevelopment() {
		r.GET("/admin/dev/info", handlers.DevInfo(envCfg, cfgManager))
	}

	// Web 管理界面 API 路由
	apiGroup := r.Group("/api")
	{
		// 子路由组（仅用于更清晰地挂载少量新路由；既有路由保持不动）
		messagesAPI := apiGroup.Group("/messages")
		responsesAPI := apiGroup.Group("/responses")
		geminiAPI := apiGroup.Group("/gemini")

		// Messages 渠道管理
		apiGroup.GET("/messages/channels", messages.GetUpstreams(cfgManager))
		apiGroup.POST("/messages/channels", messages.AddUpstream(cfgManager))
		apiGroup.PUT("/messages/channels/:id", messages.UpdateUpstream(cfgManager, channelScheduler))
		apiGroup.DELETE("/messages/channels/:id", messages.DeleteUpstream(cfgManager))
		apiGroup.POST("/messages/channels/:id/keys", messages.AddApiKey(cfgManager))
		apiGroup.DELETE("/messages/channels/:id/keys/:apiKey", messages.DeleteApiKey(cfgManager))
		apiGroup.POST("/messages/channels/:id/keys/:apiKey/top", messages.MoveApiKeyToTop(cfgManager))
		apiGroup.POST("/messages/channels/:id/keys/:apiKey/bottom", messages.MoveApiKeyToBottom(cfgManager))

		// Messages 多渠道调度 API
		apiGroup.POST("/messages/channels/reorder", messages.ReorderChannels(cfgManager))
		apiGroup.PATCH("/messages/channels/:id/status", messages.SetChannelStatus(cfgManager))
		apiGroup.POST("/messages/channels/:id/resume", handlers.ResumeChannel(channelScheduler, false))
		apiGroup.POST("/messages/channels/:id/promotion", messages.SetChannelPromotion(cfgManager))
		apiGroup.GET("/messages/channels/metrics", handlers.GetChannelMetricsWithConfig(messagesMetricsManager, cfgManager, false))
		apiGroup.GET("/messages/channels/metrics/history", handlers.GetChannelMetricsHistory(messagesMetricsManager, cfgManager, false))
		apiGroup.GET("/messages/channels/:id/keys/metrics/history", handlers.GetChannelKeyMetricsHistory(messagesMetricsManager, cfgManager, false))
		apiGroup.GET("/messages/channels/scheduler/stats", handlers.GetSchedulerStats(channelScheduler))
		apiGroup.GET("/messages/global/stats/history", handlers.GetGlobalStatsHistory(messagesMetricsManager))
		apiGroup.GET("/messages/channels/dashboard", handlers.GetChannelDashboard(cfgManager, channelScheduler))
		apiGroup.GET("/messages/ping/:id", messages.PingChannel(cfgManager))
		apiGroup.GET("/messages/ping", messages.PingAllChannels(cfgManager))

		// 缓存监控 API
		apiGroup.GET("/cache/stats", handlers.GetCacheStats(modelsResponseCache, modelsCacheMetrics))

		// Responses 渠道管理
		apiGroup.GET("/responses/channels", responses.GetUpstreams(cfgManager))
		apiGroup.POST("/responses/channels", responses.AddUpstream(cfgManager))
		apiGroup.PUT("/responses/channels/:id", responses.UpdateUpstream(cfgManager, channelScheduler))
		apiGroup.DELETE("/responses/channels/:id", responses.DeleteUpstream(cfgManager))
		apiGroup.POST("/responses/channels/:id/keys", responses.AddApiKey(cfgManager))
		apiGroup.DELETE("/responses/channels/:id/keys/:apiKey", responses.DeleteApiKey(cfgManager))
		apiGroup.POST("/responses/channels/:id/keys/:apiKey/top", responses.MoveApiKeyToTop(cfgManager))
		apiGroup.POST("/responses/channels/:id/keys/:apiKey/bottom", responses.MoveApiKeyToBottom(cfgManager))

		// Responses 多渠道调度 API
		apiGroup.POST("/responses/channels/reorder", responses.ReorderChannels(cfgManager))
		apiGroup.PATCH("/responses/channels/:id/status", responses.SetChannelStatus(cfgManager))
		apiGroup.POST("/responses/channels/:id/resume", handlers.ResumeChannel(channelScheduler, true))
		apiGroup.POST("/responses/channels/:id/promotion", handlers.SetResponsesChannelPromotion(cfgManager))
		apiGroup.GET("/responses/channels/metrics", handlers.GetChannelMetricsWithConfig(responsesMetricsManager, cfgManager, true))
		apiGroup.GET("/responses/channels/metrics/history", handlers.GetChannelMetricsHistory(responsesMetricsManager, cfgManager, true))
		apiGroup.GET("/responses/channels/:id/keys/metrics/history", handlers.GetChannelKeyMetricsHistory(responsesMetricsManager, cfgManager, true))
		apiGroup.GET("/responses/global/stats/history", handlers.GetGlobalStatsHistory(responsesMetricsManager))
		apiGroup.GET("/responses/ping/:id", responses.PingChannel(cfgManager))
		apiGroup.GET("/responses/ping", responses.PingAllChannels(cfgManager))

		// Gemini 渠道管理
		apiGroup.GET("/gemini/channels", gemini.GetUpstreams(cfgManager))
		apiGroup.POST("/gemini/channels", gemini.AddUpstream(cfgManager))
		apiGroup.PUT("/gemini/channels/:id", gemini.UpdateUpstream(cfgManager, channelScheduler))
		apiGroup.DELETE("/gemini/channels/:id", gemini.DeleteUpstream(cfgManager))
		apiGroup.POST("/gemini/channels/:id/keys", gemini.AddApiKey(cfgManager))
		apiGroup.DELETE("/gemini/channels/:id/keys/:apiKey", gemini.DeleteApiKey(cfgManager))
		apiGroup.POST("/gemini/channels/:id/keys/:apiKey/top", gemini.MoveApiKeyToTop(cfgManager))
		apiGroup.POST("/gemini/channels/:id/keys/:apiKey/bottom", gemini.MoveApiKeyToBottom(cfgManager))

		// Gemini 多渠道调度 API
		apiGroup.POST("/gemini/channels/reorder", gemini.ReorderChannels(cfgManager))
		apiGroup.PATCH("/gemini/channels/:id/status", gemini.SetChannelStatus(cfgManager))
		apiGroup.POST("/gemini/channels/:id/promotion", gemini.SetChannelPromotion(cfgManager))
		apiGroup.PUT("/gemini/loadbalance", gemini.UpdateLoadBalance(cfgManager))
		apiGroup.GET("/gemini/channels/metrics", handlers.GetGeminiChannelMetrics(geminiMetricsManager, cfgManager))
		apiGroup.GET("/gemini/channels/metrics/history", handlers.GetGeminiChannelMetricsHistory(geminiMetricsManager, cfgManager))
		apiGroup.GET("/gemini/channels/:id/keys/metrics/history", handlers.GetGeminiChannelKeyMetricsHistory(geminiMetricsManager, cfgManager))
		apiGroup.GET("/gemini/global/stats/history", handlers.GetGlobalStatsHistory(geminiMetricsManager))
		apiGroup.GET("/gemini/ping/:id", gemini.PingChannel(cfgManager))
		apiGroup.GET("/gemini/ping", gemini.PingAllChannels(cfgManager))

		// Fuzzy 模式设置
		apiGroup.GET("/settings/fuzzy-mode", handlers.GetFuzzyMode(cfgManager))
		apiGroup.PUT("/settings/fuzzy-mode", handlers.SetFuzzyMode(cfgManager))

		// 请求日志 API
		requestLogsHandler := handlers.NewRequestLogsHandler(requestLogStore)
		messagesAPI.GET("/logs", requestLogsHandler.GetLogs)
		responsesAPI.GET("/logs", requestLogsHandler.GetLogs)
		geminiAPI.GET("/logs", requestLogsHandler.GetLogs)

		// Key 熔断日志 & 重置（每个 key 仅保留 1 条熔断时日志）
		// 注意：/keys/:apiKey 已用于按 key 字符串操作；此处用 /keys/index/:keyIndex 避免 Gin 路由通配冲突
		messagesAPI.GET("/channels/:id/keys/index/:keyIndex/circuit-log", handlers.GetKeyCircuitLog(keyCircuitLogStore, cfgManager, "messages"))
		responsesAPI.GET("/channels/:id/keys/index/:keyIndex/circuit-log", handlers.GetKeyCircuitLog(keyCircuitLogStore, cfgManager, "responses"))
		geminiAPI.GET("/channels/:id/keys/index/:keyIndex/circuit-log", handlers.GetKeyCircuitLog(keyCircuitLogStore, cfgManager, "gemini"))

		messagesAPI.POST("/channels/:id/keys/index/:keyIndex/reset", handlers.ResetKeyCircuitState(channelScheduler, cfgManager, "messages"))
		responsesAPI.POST("/channels/:id/keys/index/:keyIndex/reset", handlers.ResetKeyCircuitState(channelScheduler, cfgManager, "responses"))
		geminiAPI.POST("/channels/:id/keys/index/:keyIndex/reset", handlers.ResetKeyCircuitState(channelScheduler, cfgManager, "gemini"))

		messagesAPI.POST("/channels/:id/keys/index/:keyIndex/reset-state", handlers.ResetKeyCircuitStatus(channelScheduler, cfgManager, "messages"))
		responsesAPI.POST("/channels/:id/keys/index/:keyIndex/reset-state", handlers.ResetKeyCircuitStatus(channelScheduler, cfgManager, "responses"))
		geminiAPI.POST("/channels/:id/keys/index/:keyIndex/reset-state", handlers.ResetKeyCircuitStatus(channelScheduler, cfgManager, "gemini"))

		// Key 元信息（启用/禁用）
		messagesAPI.PATCH("/channels/:id/keys/index/:keyIndex/meta", handlers.PatchAPIKeyMeta(cfgManager, "messages"))
		responsesAPI.PATCH("/channels/:id/keys/index/:keyIndex/meta", handlers.PatchAPIKeyMeta(cfgManager, "responses"))
		geminiAPI.PATCH("/channels/:id/keys/index/:keyIndex/meta", handlers.PatchAPIKeyMeta(cfgManager, "gemini"))

		// 实时请求 API
		liveRequestsHandler := handlers.NewLiveRequestsHandler(liveRequestManager)
		messagesAPI.GET("/live", liveRequestsHandler.GetLiveRequests)
		responsesAPI.GET("/live", liveRequestsHandler.GetLiveRequests)
		geminiAPI.GET("/live", liveRequestsHandler.GetLiveRequests)
	}

	// 代理端点 - Messages API
	messagesHandler := messages.NewHandler(envCfg, cfgManager, channelScheduler, billingClient, billingHandler, liveRequestManager, keyCircuitLogStore, requestLogStore)
	r.POST("/v1/messages", messagesHandler)
	r.POST("/v1/messages/count_tokens", messages.CountTokensHandler(envCfg, cfgManager, channelScheduler))

	// 代理端点 - Models API（转发到上游）
	r.GET("/v1/models", messages.ModelsHandler(envCfg, cfgManager, channelScheduler, modelsResponseCache))
	r.GET("/v1/models/:model", messages.ModelsDetailHandler(envCfg, cfgManager, channelScheduler))

	// 代理端点 - Responses API
	responsesHandler := responses.NewHandler(envCfg, cfgManager, sessionManager, channelScheduler, billingClient, billingHandler, liveRequestManager, keyCircuitLogStore, requestLogStore)
	r.POST("/v1/responses", responsesHandler)
	r.POST("/v1/responses/compact", responses.CompactHandler(envCfg, cfgManager, sessionManager, channelScheduler))

	// 代理端点 - Gemini API (原生协议)
	// 使用通配符捕获 model:action 格式，如 gemini-pro:generateContent
	// 路径格式：/v1beta/models/{model}:generateContent (Gemini 原生格式)
	geminiHandler := gemini.NewHandler(envCfg, cfgManager, channelScheduler, liveRequestManager, keyCircuitLogStore, requestLogStore)
	r.POST("/v1beta/models/*modelAction", geminiHandler)

	// 静态文件服务 (嵌入的前端)
	if envCfg.EnableWebUI {
		handlers.ServeFrontend(r, frontendFS)
	} else {
		// 纯 API 模式
		r.GET("/", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"name":    "Claude API Proxy",
				"mode":    "API Only",
				"version": "1.0.0",
				"endpoints": gin.H{
					"health": "/health",
					"proxy":  "/v1/messages",
					"config": "/admin/config/reload",
				},
				"message": "Web界面已禁用，此服务器运行在纯API模式下",
			})
		})
	}

	// 启动服务器
	addr := fmt.Sprintf(":%d", envCfg.Port)
	fmt.Printf("\n[Server-Startup] Claude API代理服务器已启动\n")
	fmt.Printf("[Server-Info] 版本: %s\n", Version)
	if BuildTime != "unknown" {
		fmt.Printf("[Server-Info] 构建时间: %s\n", BuildTime)
	}
	if GitCommit != "unknown" {
		fmt.Printf("[Server-Info] Git提交: %s\n", GitCommit)
	}
	fmt.Printf("[Server-Info] 管理界面: http://localhost:%d\n", envCfg.Port)
	fmt.Printf("[Server-Info] API 地址: http://localhost:%d/v1\n", envCfg.Port)
	fmt.Printf("[Server-Info] Claude Messages: POST /v1/messages\n")
	fmt.Printf("[Server-Info] Codex Responses: POST /v1/responses\n")
	fmt.Printf("[Server-Info] Gemini API: POST /v1beta/models/{model}:generateContent\n")
	fmt.Printf("[Server-Info] Gemini API: POST /v1beta/models/{model}:streamGenerateContent\n")
	fmt.Printf("[Server-Info] 健康检查: GET /health\n")
	fmt.Printf("[Server-Info] 环境: %s\n", envCfg.Env)
	// 计费模式提示
	if envCfg.IsBillingEnabled() {
		fmt.Printf("[Server-Info] 计费模式: 已启用 (swe-agent: %s)\n", envCfg.SweAgentBillingURL)
	} else {
		fmt.Printf("[Server-Info] 计费模式: 未启用 (使用单用户模式)\n")
	}
	// 检查是否使用默认密码，给予提示
	if envCfg.ProxyAccessKey == "your-proxy-access-key" {
		fmt.Printf("[Server-Warn] 访问密钥: your-proxy-access-key (默认值，建议通过 .env 文件修改)\n")
	}
	fmt.Printf("\n")

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 用于传递关闭结果
	shutdownDone := make(chan struct{})

	// 优雅关闭：监听系统信号
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		signal.Stop(sigChan) // 停止信号监听，避免资源泄漏

		log.Println("[Server-Shutdown] 收到关闭信号，正在优雅关闭服务器...")

		// 创建超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[Server-Shutdown] 警告: 服务器关闭时发生错误: %v", err)
		} else {
			log.Println("[Server-Shutdown] 服务器已安全关闭")
		}

		// 关闭价格表服务
		if pricingService != nil {
			pricingService.Stop()
			log.Println("[Pricing-Shutdown] 价格表服务已关闭")
		}

		close(shutdownDone)
	}()

	// 启动服务器（阻塞直到关闭）
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务器启动失败: %v", err)
	}

	// 等待关闭完成（带超时保护，避免死锁）
	select {
	case <-shutdownDone:
		// 正常关闭完成
	case <-time.After(15 * time.Second):
		log.Println("[Server-Shutdown] 警告: 等待关闭超时")
	}
}
