# 任务清单: 同步 upstream 至 v2.5.1（claude-proxy）

目录: `helloagents/plan/202601191056_sync_upstream_v251/`

---

## 1. 基线与差异确认（只读）
- [√] 1.1 确认本仓库当前基线与自研改动清单（对照 `v2.4.32`）
- [√] 1.2 拉取 upstream tags 并生成 `v2.4.32..v2.4.34..v2.5.1` 的差异摘要（文件/commit 级）

## 2. 后端同步（对齐 upstream v2.4.34）
- [√] 2.1 在 `backend-go/internal/handlers/common/request.go` 对齐 `ExtractConversationID`：新增 `X-Gemini-Api-Privileged-User-Id` 优先级支持，补单测
- [√] 2.2 在 `backend-go/internal/middleware/auth.go` 支持 `x-goog-api-key`，并覆盖 Proxy/Billing 两种鉴权路径的测试
- [√] 2.3 在 `backend-go/internal/metrics/channel_metrics.go` 引入 `GetRecentActivityMultiURL` 与数据结构，补单测（窗口边界/多 URL×Key 聚合/空输入）
- [√] 2.4 在 `backend-go/internal/handlers/channel_metrics_handler.go` 的 dashboard 返回中加入 `recentActivity`（messages/responses），补 handler 测试
- [√] 2.5 新增 `backend-go/internal/handlers/gemini/dashboard.go` 并在 `backend-go/main.go` 注册 `/api/gemini/channels/dashboard`，补 handler 测试
- [√] 2.6 对齐 `backend-go/internal/converters/gemini_converter.go` 的 `thought_signature` 兼容处理，补/对齐 converter 测试

## 3. 前端一次性同步到 upstream v2.5.1（Router + Pinia）
- [√] 3.1 引入并接入 `vue-router` 与 `pinia`（对齐 upstream 的 `frontend/src/main.ts` 与 `frontend/src/router/*`、`frontend/src/stores/*`）
- [√] 3.2 对齐 `frontend/src/services/api.ts`：Dashboard 数据按 Tab 缓存、错误处理改进
- [√] 3.3 对齐 `frontend/src/views/ChannelsView.vue` 与 `frontend/src/components/ChannelOrchestration.vue`（活跃度颜色与缓存逻辑）
- [√] 3.4 清理 Tailwind/DaisyUI：更新 `frontend/src/assets/style.css`，移除不再使用的配置/依赖（按 upstream）

## 4. 迁移本仓库自研功能到新前端架构
- [√] 4.1 将“请求监控页/组件”迁移为路由页（路径待确认），保留入口按钮与与 API type 联动
- [√] 4.2 校验 KeyMeta enable/disable + description 的 UI/数据流在新 store 架构下不回退

## 5. 仓库文件与版本对齐
- [√] 5.1 更新 `VERSION` 为 `v2.5.1` 并同步 `CHANGELOG.md`（必要时补充本仓库自研差异说明）
- [√] 5.2 评估并引入 upstream `.gitattributes` 与 `.gitignore` 调整（按你确认的 lock file 策略）

## 6. 安全检查
- [√] 6.1 执行安全检查（鉴权头扩展不引入绕过、日志脱敏、请求/响应日志开关在生产默认安全）

## 7. 测试与构建验证
- [√] 7.1 后端：运行 `cd backend-go && make test`，确保新增/回归测试通过
- [√] 7.2 前端：运行 `cd frontend && bun run build`（或项目既定构建命令），确保产物可 embed
- [√] 7.3 集成：运行 `make build` / `make run` 做最小冒烟（健康检查、Web UI、三个代理端点）

---

## 任务状态符号
- `[ ]` 待执行
- `[√]` 已完成
- `[X]` 执行失败
- `[-]` 已跳过
- `[?]` 待确认
