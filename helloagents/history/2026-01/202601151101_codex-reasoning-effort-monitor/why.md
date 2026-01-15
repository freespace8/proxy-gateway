# 变更提案: 请求监控展示 Codex 思考等级（reasoning.effort）

## 需求背景
目前 `/monitor` 的实时请求与请求日志只展示模型/渠道等信息，无法看到调用方在 `/v1/responses` 请求中设置的 `reasoning.effort`（如 `xhigh`）。排障与使用分析时，需要在同一视图快速对齐“模型 + 思考等级”。

## 产品分析

### 目标用户与场景
- **用户群体:** 管理端使用者（运维/开发/排障人员）
- **使用场景:** 排查响应慢/成本高/效果波动时，确认是否因思考档位变化导致
- **核心痛点:** 需要翻日志或抓包才能知道 `reasoning.effort`，效率低

### 价值主张与成功指标
- **价值主张:** 在请求监控中直观看到 Codex 思考档位，降低排障与分析成本
- **成功指标:** 请求日志列表与实时请求列表能稳定显示该字段；非 Codex 渠道为空

### 人文关怀
`reasoning.effort` 仅为档位字符串，不涉及用户内容；实现中避免额外输出原始请求体，降低敏感信息暴露风险。

## 变更内容
1. 后端对 `/v1/responses` 解析并记录 `reasoning.effort`（优先“实际上游请求体”，缺失则回退入口原始请求体）。
2. 前端请求监控：
   - 请求日志列表：在“模型”列后新增“思考”列。
   - 实时请求列表：模型名称后追加显示思考等级。
3. Claude/Gemini 渠道该字段为空即可。
4. 请求日志固定使用内存存储，不依赖 SQLite。

## 影响范围
- **模块:** `backend-go/internal/handlers/*`、`backend-go/internal/providers/*`、`backend-go/internal/metrics/*`、`backend-go/internal/monitor/*`、`frontend/src/*`
- **文件:**
  - `backend-go/internal/handlers/responses/handler.go`
  - `backend-go/internal/providers/responses.go`
  - `backend-go/internal/metrics/request_log.go`
  - `backend-go/internal/monitor/live_requests.go`
  - `backend-go/internal/handlers/messages/handler.go`
  - `backend-go/internal/handlers/gemini/handler.go`
  - `frontend/src/services/api.ts`
  - `frontend/src/components/RequestLogList.vue`
  - `frontend/src/components/LiveRequestMonitor.vue`
- **API:**
  - `GET /api/{messages|responses|gemini}/logs`（新增可选字段 `reasoningEffort`）
  - `GET /api/{messages|responses|gemini}/live`（新增可选字段 `reasoningEffort`）
- **数据:** 请求日志固定为内存环形缓冲，不落库；容量由 `REQUEST_LOGS_MEMORY_MAX_SIZE` 控制。

## 核心场景

### 需求: 查看 Codex 思考等级
**模块:** 请求监控（/monitor）

#### 场景: 请求体包含 reasoning.effort
当调用方请求体包含：
- `reasoning.effort="xhigh"`（或其他档位）

预期：
- “请求日志”列表在模型列后显示 `xhigh`
- “实时请求”列表在模型名后显示 `xhigh`

### 需求: 以实际上游请求体为准
**模块:** Responses 上游请求构建

#### 场景: 入口为 minimal，但上游被归一化为 low
当系统对上游请求做兼容处理导致档位变化（如 `minimal -> low`）时，预期监控展示 `low`。

### 需求: 其他 API 类型为空
**模块:** 请求监控（/monitor）

#### 场景: messages/gemini
预期：
- “思考”字段为空（或显示占位符），不影响现有展示。

## 风险评估
- **风险:** `reasoning.effort` 缺失/非字符串/格式异常导致展示错误
- **缓解:** 解析失败即置空；仅在非空时覆盖；全链路保持向后兼容（新增字段不影响旧客户端）
