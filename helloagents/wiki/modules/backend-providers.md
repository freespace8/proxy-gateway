# 后端-Providers（backend-go/internal/providers）

## 目的
对接各类上游（Claude/OpenAI/Gemini 等）的请求/响应适配。

## 模块概述
- **职责:** 构建上游请求、处理流式与非流式响应、错误映射与必要的协议差异处理。
- **状态:** ✅稳定
- **最后更新:** 2026-01-20

## 规范
- 上游认证头设置必须与 upstream 类型匹配；请求头透传需过滤 hop-by-hop 头。
- 多上游/多 baseUrl 场景避免在请求对象上做原地修改导致污染。

## 依赖
- `backend-go/internal/utils`
- `backend-go/internal/types`

## 变更历史
- （暂无）

