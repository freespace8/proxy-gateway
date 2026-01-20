# API 手册

## 概述
本项目提供两类 API：
- **代理 API**：对外提供 `/v1/messages`、`/v1/responses`、`/v1beta/models/*` 等端点的转发/转换。
- **管理 API**：对外提供 `/api/*` 的渠道管理、仪表盘、监控等能力。

## 认证方式
- **统一鉴权**：Header `x-api-key` 或 `Authorization: Bearer <key>`
- **兼容**：Gemini SDK 场景可支持 `x-goog-api-key`（仅用于鉴权兼容，不等价于上游 key）

---

## 接口列表（摘要）

### 代理端点

#### [POST] /v1/messages
**描述:** Claude Messages API 代理（统一鉴权、多渠道调度）。

#### [POST] /v1/responses
**描述:** Responses API 代理（会话/多轮上下文）。

#### [POST] /v1beta/models/{model}:generateContent
**描述:** Gemini API 代理（可选协议转换/多渠道调度）。

#### [POST] /v1beta/models/{model}:streamGenerateContent
**描述:** Gemini SSE 流式代理。

### 管理端点（示例）

#### [GET] /api/*/channels
**描述:** 获取渠道列表（按 API 类型分组）。

#### [GET] /api/*/channels/dashboard
**描述:** 渠道仪表盘（channels + metrics + stats + recentActivity）。

> Gemini 渠道字段补充：`injectDummyThoughtSignature` / `stripThoughtSignature`。
