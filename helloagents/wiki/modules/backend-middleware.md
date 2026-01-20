# 后端-Middleware（backend-go/internal/middleware）

## 目的
提供鉴权、日志等跨切面能力。

## 模块概述
- **职责:** ProxyAuth/WebAuth、请求日志、CORS 与通用中间件。
- **状态:** ✅稳定
- **最后更新:** 2026-01-20

## 规范
- 生产环境默认最小日志；避免记录敏感信息。

## 依赖
- `backend-go/internal/config`

## 变更历史
- （暂无）

