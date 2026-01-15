# 任务清单: 请求监控展示 Codex 思考等级（reasoning.effort）

目录: `helloagents/plan/202601151101_codex-reasoning-effort-monitor/`

---

## 0. 需求确认
- [√] 0.1 实时请求“模型后追加”展示样式使用：`model (effort)`
- [√] 0.2 请求日志固定内存存储，不使用 SQLite

## 1. 后端：记录 reasoningEffort
- [√] 1.1 在 `backend-go/internal/metrics/request_log.go` 为 `RequestLogRecord` 增加 `ReasoningEffort` 字段，并在三类 handler 写日志时带上该字段
- [√] 1.2 在 `backend-go/internal/monitor/live_requests.go` 为 `LiveRequest` 增加 `ReasoningEffort` 字段，并在三类 handler 的 `updateLive()` 中带上该字段
- [√] 1.3 在 `backend-go/internal/providers/responses.go` 从实际上游请求体解析 `reasoning.effort` 写入 gin context；在 `backend-go/internal/handlers/responses/handler.go` 于 `ConvertToProviderRequest` 后读取并“非空覆盖”入口兜底值
- [√] 1.4 在 `backend-go/main.go` 固定使用 `metrics.NewMemoryRequestLogStore` 作为 requestLogStore（不走 SQLite）

## 2. 前端：请求监控 UI 展示
- [√] 2.1 在 `frontend/src/services/api.ts` 为 `RequestLogRecord`/`LiveRequest` 增加可选字段 `reasoningEffort?: string`
- [√] 2.2 在 `frontend/src/components/RequestLogList.vue` 增加“思考”列（位于“模型”列后），空值显示 `--`
- [√] 2.3 在 `frontend/src/components/LiveRequestMonitor.vue` 模型名称后追加显示思考等级（按 0.1 确认的样式），空值不展示

## 3. 安全检查
- [√] 3.1 检查仅解析并展示 `reasoning.effort`，不新增原始请求体日志输出，不引入敏感信息透出

## 4. 测试
- [√] 4.1 后端：`cd backend-go && go test ./...`（或 `make test`）验证通过
- [√] 4.2 前端：`cd frontend && npm run type-check`（环境无 bun），手动打开 `/monitor` 验证请求日志与实时请求均符合预期

---

## 任务状态符号
- `[ ]` 待执行
- `[√]` 已完成
- `[X]` 执行失败
- `[-]` 已跳过
- `[?]` 待确认
