# 变更提案: 同步 upstream 至 v2.5.1（claude-proxy）

## 需求背景
当前仓库基线为 `v2.4.32`，且已存在多处自研增强（请求监控/请求日志、Key 元信息 enable/disable + description、去除 SQLite 指标持久化等）。

upstream `BenedictKing/claude-proxy` 在 `v2.4.34` 与 `v2.5.1` 引入了一批后端协议兼容与前端架构重构：
- 后端：Gemini 会话亲和 Header 支持、Gemini SDK 鉴权头支持、Gemini Dashboard 聚合接口、渠道 15 分钟实时活跃度聚合（recentActivity）、Gemini function call `thought_signature` 兼容性修复等。
- 前端：去 Tailwind/DaisyUI、引入 Vue Router + Pinia 并重构状态管理与页面结构，改进 API 错误处理与闪烁问题。

本需求目标：在不回退现有自研能力的前提下，**将 upstream 的上述变更吸收到本项目**，并将版本推进至 `v2.5.1`。

## 变更内容
1. 同步后端 `v2.4.34` 关键能力（recentActivity、Gemini Dashboard、鉴权头支持、会话 ID 提取增强、Gemini converter 修复等）。
2. 同步前端至 `v2.5.1` 的 Router + Pinia 架构，并迁移本仓库的“请求监控页”等自研页面/入口到新架构中。
3. 对齐构建与仓库文件规范（`.gitattributes`、lock file 策略、`.gitignore` 调整等，按需）。

## 影响范围
- **模块:**
  - `backend-go/internal/handlers/*`
  - `backend-go/internal/metrics/*`
  - `backend-go/internal/middleware/*`
  - `backend-go/internal/converters/*`
  - `frontend/src/*`
  - 根目录版本与仓库配置文件（`VERSION`、`.gitattributes`、`.gitignore`）
- **文件:**
  - 预计涉及数十个文件（前端重构为主）
- **API:**
  - 新增/调整：`/api/gemini/channels/dashboard`（与 messages/responses dashboard 对齐）
  - 扩展鉴权 Header：`x-goog-api-key`
  - 会话标识提取新增 Header：`X-Gemini-Api-Privileged-User-Id`
- **数据:**
  - 指标数据结构新增：`recentActivity`（15m，150 段×6s，聚合多 BaseURL×Key）
  - 前端状态管理迁移到 Pinia（仅本地状态/偏好持久化）

## 核心场景

### 需求: 同步 upstream 到 v2.5.1
**模块:** repo / backend-go / frontend
将 upstream 的增量按“后端能力 → 前端架构 → 兼容自研能力”的顺序同步，最终版本推进到 `v2.5.1`。

#### 场景: 作为管理员打开 Web UI
在新前端架构下仍可正常鉴权、切换 Messages/Responses/Gemini，页面数据加载稳定（无 Tab 切换闪烁回退）。
- 预期：登录/自动认证流程不变，主要页面正常展示、关键交互可用。

#### 场景: 调用 Gemini API（SDK 兼容）
支持 Gemini SDK 使用 `x-goog-api-key` 携带访问密钥进行统一鉴权。
- 预期：不改变现有 `x-api-key` / `Authorization: Bearer` 行为，仅增加兼容头。

#### 场景: 会话亲和（Gemini header 参与）
Responses/Gemini 等路径在提取对话标识时支持 `X-Gemini-Api-Privileged-User-Id`，提升会话亲和性一致性。
- 预期：Conversation id 提取优先级符合 upstream 约定。

#### 场景: 渠道概览展示最近 15 分钟活跃度
Dashboard API 返回 `recentActivity` 并在 UI 中可视化，支持多 BaseURL / 多 Key 聚合。
- 预期：活跃度柱状/波形实时更新，颜色与成功率一致，且不会造成性能明显下降。

#### 场景: 继续使用本仓库的“请求监控”页面
引入 Router + Pinia 后，仍能访问本仓库自研的请求监控视图，并与新路由结构共存。
- 预期：入口清晰，路由/状态与 upstream 页面不冲突。

## 风险评估
- **风险:** 前端结构性重构（Router/Pinia）导致现有自研页面/状态逻辑迁移成本高，且存在回归风险。
- **缓解:** 采用“先对齐 upstream 再迁移自研功能”的两阶段策略；为关键路径补充前端/后端回归用例；升级过程分批提交并在每批后运行构建与测试。

