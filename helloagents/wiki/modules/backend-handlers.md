# 后端-Handlers（backend-go/internal/handlers）

## 目的
承载 HTTP 路由：代理端点与管理端点。

## 模块概述
- **职责:** 路由与参数解析、鉴权、调用调度器、透传上游响应、输出统一错误结构。
- **状态:** ✅稳定
- **最后更新:** 2026-01-20

## 规范
- 代理端点统一走代理鉴权；管理端点统一走管理鉴权（以项目实际实现为准）。
- 流式响应必须正确处理连接关闭与 header。
- Gemini 代理支持 Gemini CLI 工具字段兼容，并可按渠道配置对请求的 `thoughtSignature` 做注入/移除（仅在 gemini 上游转发时生效）。

## 依赖
- `backend-go/internal/config`
- `backend-go/internal/scheduler`
- `backend-go/internal/providers`
- `backend-go/internal/metrics`
- `backend-go/internal/middleware`

## 变更历史
- [202601201123_sync_upstream_v2.5.6_gemini_cli](../../history/2026-01/202601201123_sync_upstream_v2.5.6_gemini_cli/) - Gemini CLI 兼容与 thoughtSignature 注入/移除
