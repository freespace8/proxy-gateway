# 后端-Session（backend-go/internal/session）

## 目的
提供 Responses API 会话与 Trace 亲和相关能力。

## 模块概述
- **职责:** 管理会话上下文、Trace 亲和映射与 TTL。
- **状态:** ✅稳定
- **最后更新:** 2026-01-20

## 规范
- 亲和与会话必须可过期；避免无限增长。

## 依赖
- `backend-go/internal/types`

## 变更历史
- （暂无）

