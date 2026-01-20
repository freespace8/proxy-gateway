# proxy-gateway（claude-proxy fork）

> 本文件包含项目级别的核心信息。详细的模块文档见 `modules/` 目录。

---

## 1. 项目概述

### 目标与背景
提供统一的 Claude / Codex / Gemini API 代理网关：统一鉴权、统一入口、多渠道调度、Key 管理与可视化面板。

### 范围
- **范围内:** 代理转发（Messages/Responses/Gemini）、渠道与 Key 管理、调度与熔断、监控与指标、单容器一体化部署。
- **范围外:** 作为上游厂商 SDK 的完全替代；复杂网关策略（可在上游/前置 LB 层实现）。

### 干系人
- **负责人:** 未指定（以仓库维护者/贡献者为准）

---

## 2. 模块索引

| 模块名称 | 职责 | 状态 | 文档 |
|---------|------|------|------|
| 后端-配置 | 运行时配置、渠道/Key 更新与持久化 | ✅稳定 | [modules/backend-config.md](modules/backend-config.md) |
| 后端-Handlers | HTTP 路由与管理 API/代理端点 | ✅稳定 | [modules/backend-handlers.md](modules/backend-handlers.md) |
| 后端-Middleware | 鉴权/日志/通用中间件 | ✅稳定 | [modules/backend-middleware.md](modules/backend-middleware.md) |
| 后端-Providers | 上游适配与透传/转换 | ✅稳定 | [modules/backend-providers.md](modules/backend-providers.md) |
| 后端-Converters | 协议转换（Gemini↔Claude/OpenAI 等） | ✅稳定 | [modules/backend-converters.md](modules/backend-converters.md) |
| 后端-Scheduler | 多渠道调度、Trace 亲和 | ✅稳定 | [modules/backend-scheduler.md](modules/backend-scheduler.md) |
| 后端-Metrics | 渠道健康/统计、请求日志/熔断记录 | ✅稳定 | [modules/backend-metrics.md](modules/backend-metrics.md) |
| 后端-Session | Responses 会话与 Trace 亲和 | ✅稳定 | [modules/backend-session.md](modules/backend-session.md) |
| 后端-Monitor | 正在进行请求（live requests）监控 | ✅稳定 | [modules/backend-monitor.md](modules/backend-monitor.md) |
| 后端-Utils | 通用工具（header、mask、stream 等） | ✅稳定 | [modules/backend-utils.md](modules/backend-utils.md) |
| 前端-管理台 | 渠道编排/配置/监控页面 | ✅稳定 | [modules/frontend-ui.md](modules/frontend-ui.md) |
| 构建与部署 | Make/Docker/发布产物 | ✅稳定 | [modules/build-deploy.md](modules/build-deploy.md) |

---

## 3. 快速链接
- [技术约定](../project.md)
- [架构设计](arch.md)
- [API 手册](api.md)
- [数据模型](data.md)
- [变更历史](../history/index.md)

