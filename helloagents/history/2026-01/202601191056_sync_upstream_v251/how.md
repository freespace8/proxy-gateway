# 技术设计: 同步 upstream 至 v2.5.1（claude-proxy）

## 技术方案

### 核心技术
- 后端：Go + Gin（保持现有结构），对齐 upstream `v2.4.34/v2.5.1` 的 handlers/metrics/middleware/converters 变更。
- 前端：Vue 3 + Vuetify，升级为 upstream 的 Vue Router + Pinia 架构；迁移本仓库自研页面到路由体系。

### 实现要点
- **同步策略（分层合入，避免冲突扩大）**
  1) 先同步后端 `v2.4.34`：新增 `recentActivity` 结构/聚合算法、Gemini dashboard 端点、鉴权头 `x-goog-api-key`、会话 ID Header 扩展、Gemini converter 修复与测试。
  2) 再同步前端到 `v2.5.1`：引入 router/pinia/stores/views，并逐项迁移本仓库自研入口（请求监控页、监控按钮等）。
  3) 最后对齐版本号与仓库文件：`VERSION`、`.gitattributes`、lock file 策略、`.gitignore`。

- **后端关键改动点（来自 upstream v2.4.34）**
  - `backend-go/internal/metrics/channel_metrics.go`：新增 `GetRecentActivityMultiURL` 与数据结构，聚合多 BaseURL×Key 的 15 分钟活跃度。
  - `backend-go/internal/handlers/channel_metrics_handler.go`：Messages/Responses 的 dashboard 返回新增 `recentActivity` 字段。
  - 新增 `backend-go/internal/handlers/gemini/dashboard.go` 并在 `backend-go/main.go` 注册 `/api/gemini/channels/dashboard`：一次返回 channels/metrics/stats/recentActivity。
  - `backend-go/internal/handlers/common/request.go`：`ExtractConversationID` 新增对 `X-Gemini-Api-Privileged-User-Id` 的支持（优先级插入到 Session_id 与 prompt_cache_key 之间）。
  - `backend-go/internal/middleware/auth.go`：`getAPIKey` 新增 `x-goog-api-key` 头支持（与现有 `x-api-key`/`Authorization` 并列）。
  - `backend-go/internal/converters/gemini_converter.go`：对齐 Gemini function call `thought_signature` 字段透传/兼容处理（必要时补测）。

- **前端关键改动点（来自 upstream v2.5.1）**
  - 引入 `vue-router`：路由结构 `/:channels/:type`（messages/responses/gemini）并默认重定向到 messages。
  - 引入 `pinia` + `pinia-plugin-persistedstate`：拆分 AuthStore/ChannelStore/SystemStore/PreferencesStore/DialogStore。
  - `frontend/src/services/api.ts`：对齐错误处理与响应解析；将 dashboard 数据按 Tab 进行缓存，避免切换闪烁。
  - `frontend/src/components/ChannelOrchestration.vue`：活跃度颜色逻辑优化与缓存。
  - 移除 Tailwind/DaisyUI：清理 `style.css` 的 `@tailwind` 指令与相关配置/依赖（与 Vuetify 统一）。

## 架构决策 ADR

### ADR-001: 前端是否完全对齐 upstream Router + Pinia（已选）
**上下文:** upstream `v2.5.1` 进行了结构性重构；本仓库目前无 router/pinia，且存在自研页面入口（请求监控）。
**决策:** 采用 upstream Router + Pinia 作为主干架构，随后将自研页面以“新增路由/新增 store（如需要）”方式并入。
**理由:** 一次性对齐 upstream 后，后续同步成本更低；状态管理集中可降低长期维护成本。
**替代方案:** 仅同步后端或仅局部引入（保留现有 App.vue 手写路由） → 拒绝原因: 长期漂移更大，后续合并成本更高。
**影响:** 短期改动面大，必须补充回归与构建验证；迁移过程需明确“自研功能保留清单”。

## API设计

### GET /api/gemini/channels/dashboard
- **请求:** 无（可扩展 query：如 duration/interval 但建议与 messages/responses 保持一致的最小接口）
- **响应:** `{ channels, loadBalance, metrics, stats, recentActivity }`

## 数据模型
- `recentActivity`: 150 段×6 秒（15 分钟），每段包含 request/success/failure/inputTokens/outputTokens，并给出 15m 平均 RPM/TPM。

## 安全与性能
- **安全:**
  - `x-goog-api-key` 仅作为访问密钥的额外承载方式，仍严格比对 `PROXY_ACCESS_KEY`/计费模式鉴权逻辑。
  - 不引入新的外部鉴权绕过路径；日志中继续脱敏 key。
- **性能:**
  - `recentActivity` 聚合需在 MetricsManager 读锁内扫描历史：注意控制窗口与数据结构大小，避免高频刷新导致 CPU 增长。
  - 前端活跃度渲染采用 upstream 的缓存策略（computed cache）降低重复计算。

## 测试与部署
- **测试:**
  - 后端：对齐 upstream 新增的 metrics/activity、gemini dashboard、conversation id 提取、auth header 的单元测试（表驱动 + httptest）。
  - 前端：保持轻量（现有无统一测试框架时按既有方式补关键 utils/component 的最小用例）。
- **部署:**
  - 完成同步后 bump `VERSION` 为 `v2.5.1`；按现有 `make build` 产物策略构建并验证 Docker 镜像可启动。

## 待确认（执行前必须确认）
1. 是否接受 **新增** `x-goog-api-key` 作为代理统一鉴权头？（默认建议接受）
2. lock file 策略：是否同时提交 `frontend/package-lock.json`（npm）与 `frontend/bun.lock`（bun），还是仅保留 bun？
3. 是否严格对齐 upstream `.gitignore`（取消忽略 lock 文件等），还是按本仓库习惯保守合并？
4. “请求监控页”在新路由中的落点：`/monitor` 独立页，还是合并为 `/channels/:type/monitor`？

