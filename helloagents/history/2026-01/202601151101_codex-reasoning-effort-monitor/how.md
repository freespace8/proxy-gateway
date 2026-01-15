# 技术设计: 请求监控展示 Codex 思考等级（reasoning.effort）

## 技术方案

### 核心技术
- Go（Gin）+ `gjson`：从 JSON bytes 快速提取 `reasoning.effort`
- 前后端通过新增可选字段 `reasoningEffort` 做展示（向后兼容）

### 实现要点
- **入口解析（fallback）**：在 `backend-go/internal/handlers/responses/handler.go` 读取到 `bodyBytes` 后，从入口原始请求体解析 `reasoning.effort`，写入 `reqCtx.reasoningEffort`，用于兜底展示。
- **实际上游解析（优先）**：在 `backend-go/internal/providers/responses.go` 构建实际上游请求体 `reqBody` 后，从 `reqBody` 解析 `reasoning.effort`，通过 `gin.Context.Set` 写入一个上下文 key；在 handler 中 `ConvertToProviderRequest` 返回后读取该 key，若非空则覆盖 `reqCtx.reasoningEffort`。
- **请求日志**：在 `metrics.RequestLogRecord` 增加 `ReasoningEffort` 字段，并在三类 handler 写请求日志时带上：
  - responses：填充（上述解析结果）
  - messages / gemini：留空
- **实时请求**：在 `monitor.LiveRequest` 增加 `ReasoningEffort` 字段；三类 handler 的 `updateLive()` 将该字段带上（responses 填充，其他留空）。
- **日志存储策略**：请求日志（`/api/{type}/logs`）固定使用内存环形缓冲，不写入 SQLite。

## 架构决策 ADR
### ADR-001: 解析口径采用“上游优先，入口兜底”
**上下文:** 用户选择 2/4：三 tab 均展示字段；口径优先实际上游请求体。
**决策:** 以 `providers/responses.go` 生成的实际上游 `reqBody` 为准；若上游不包含该字段则回退入口原始请求体。
**理由:** 能反映实际发送给上游的档位（包含归一化/兼容逻辑），同时避免转换模式丢字段导致完全缺失。
**替代方案:** 仅解析入口请求体 → 拒绝原因: 无法反映上游归一化后的实际值。
**影响:** 需要在 provider 与 handler 之间通过 gin context 传递一个字符串字段，改动可控且局部。

## API设计
### GET /api/{messages|responses|gemini}/logs
- **响应新增字段:** `logs[].reasoningEffort?: string`

### GET /api/{messages|responses|gemini}/live
- **响应新增字段:** `requests[].reasoningEffort?: string`

## 数据模型
请求日志固定为内存环形缓冲（`metrics.MemoryRequestLogStore`），不落库；无需 SQLite 迁移。

## 安全与性能
- **安全:** 仅解析 `reasoning.effort` 字符串，不记录原始 body；避免增加敏感信息暴露面。
- **性能:** `gjson.GetBytes` O(n) 解析一次；字段很短，对请求链路影响可忽略。

## 测试与部署
- **测试:**
  - 后端：补充单测覆盖入口兜底、上游优先覆盖、logs/live API 返回字段（responses 有值，messages/gemini 为空）。
  - 前端：`bun run type-check`，手动打开 `/monitor` 验证两处 UI。
- **部署:** 无配置变更；上线后即可在 `/monitor` 看到新增字段（取决于请求是否携带 `reasoning.effort`）。
