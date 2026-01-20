# 后端-Converters（backend-go/internal/converters）

## 目的
提供协议转换能力（例如 Gemini ↔ Claude/OpenAI）。

## 模块概述
- **职责:** 请求/响应结构映射；工具调用结构转换；使用量字段归一化。
- **状态:** ✅稳定
- **最后更新:** 2026-01-20

## 规范
- 转换需保留尽可能多的语义信息；遇到无法表达的字段需有明确降级策略。

## 依赖
- `backend-go/internal/types`

## 变更历史
- （暂无）

