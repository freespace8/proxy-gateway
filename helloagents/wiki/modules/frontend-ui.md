# 前端-管理台（frontend/）

## 目的
提供可视化管理界面：渠道配置、调度状态与监控。

## 模块概述
- **职责:** 渠道编排、添加/编辑渠道、指标/监控视图、调用后端管理 API。
- **状态:** ✅稳定
- **最后更新:** 2026-01-20

## 规范
- 任何涉及密钥的展示必须脱敏；输入框默认 password。
- 字段命名与后端 JSON 字段保持一致。
- Gemini 渠道支持 `injectDummyThoughtSignature` / `stripThoughtSignature` 两个开关（添加/编辑渠道时可配置）。

## 依赖
- Vuetify / Vue Router / Pinia

## 变更历史
- [202601201123_sync_upstream_v2.5.6_gemini_cli](../../history/2026-01/202601201123_sync_upstream_v2.5.6_gemini_cli/) - Gemini 渠道新增 thoughtSignature 开关
