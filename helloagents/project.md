# 项目技术约定

## 技术栈
- **后端:** Go 1.22+ / Gin
- **前端:** Vue 3 / Vite / Vuetify 3
- **构建:** Make / Bun（前端）

## 开发约定
- **Go:** `gofmt` 必跑；尽量保持包职责单一（`backend-go/internal/*`）。
- **前端:** TypeScript strict；遵循现有 Vuetify 风格与组件约定。
- **配置/密钥:** 仅提交示例文件（如 `*.example`），禁止提交真实密钥。

## 错误与日志
- **策略:** 代理端点错误尽量透传上游；管理端点返回 JSON 错误信息。
- **日志:** 生产环境建议关闭详细请求/响应日志；避免记录明文 API Key。

## 测试与流程
- **测试:** 后端优先表驱动 + `httptest`；修改关键逻辑尽量补 `_test.go`。
- **提交:** 约定使用 Conventional Commits 风格（如 `feat(x): ...` / `fix(x): ...`）。

