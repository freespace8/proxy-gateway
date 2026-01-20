# 后端-配置（backend-go/internal/config）

## 目的
提供运行时配置加载、校验、更新与持久化能力。

## 模块概述
- **职责:** 维护 `.config/config.json`；提供渠道/Key 管理与更新方法；提供默认值与兼容逻辑。
- **状态:** ✅稳定
- **最后更新:** 2026-01-20

## 规范
- 配置结构以 Go struct 为准；更新操作需考虑并发读写与向后兼容。
- 禁止在日志/返回中输出明文 API Key。
- Gemini 渠道可选字段：`injectDummyThoughtSignature` / `stripThoughtSignature`（默认 false）。

## 依赖
- `backend-go/internal/utils`

## 变更历史
- [202601201123_sync_upstream_v2.5.6_gemini_cli](../../history/2026-01/202601201123_sync_upstream_v2.5.6_gemini_cli/) - 增加 Gemini thoughtSignature 相关配置字段
