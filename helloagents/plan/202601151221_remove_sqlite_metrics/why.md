# 变更提案: 去除 sqlite 指标持久化（纯内存保留 7 天）

## 需求背景
- 当前后端使用 `.config/metrics.db`(sqlite) 持久化指标与 key 熔断日志；本地使用场景下希望减少磁盘 I/O/依赖以提升转发性能与稳定性。
- 允许重启丢失历史数据；但希望管理界面现有“全局/渠道/Key 历史图表 + Key 熔断日志查询”能力仍可用，仅保留最近 7 天即可。

## 变更内容
1. 移除 sqlite 指标持久化实现与依赖（`modernc.org/sqlite`、`metrics.SQLiteStore`、daily_stats 聚合任务）。
2. 指标历史与聚合全部改为内存保留 7 天（复用现有 in-memory 路径，将保留窗口从 24h 扩展到 7d）。
3. Key 熔断“最后一次失败日志”改为内存保存（按 key_id 覆盖更新），超出 7 天自动过期。
4. API 行为保持兼容：当请求 duration > 7 天时，返回 7 天数据并携带 warning（沿用现有响应字段）。

## 影响范围
- **模块:** `backend-go/internal/metrics/`、`backend-go/main.go`、`backend-go/internal/handlers/`、`backend-go/internal/handlers/common/`
- **API:** 不新增路径；历史接口的“降级/截断”策略从 24h/DB 变为 7d/内存；Key circuit-log 在无数据时仍返回 404
- **数据:** 不再读写 `.config/metrics.db`（可保留为无用文件；如需可手动删除）

## 核心场景

### 需求: 指标历史（7 天）
**模块:** metrics + handlers

#### 场景: 查看全局/渠道/Key 历史（duration=today/24h/7d/30d）
- 返回数据点与汇总；当请求 duration > 7 天时，仅返回最近 7 天并给出 warning

### 需求: Key 熔断日志
**模块:** handlers/common + handlers/key_circuit_log_handler.go

#### 场景: 查看某 key 的“最后一次失败日志”
- 7 天内可查；超过 7 天自动过期为“无日志”

## 风险评估
- **风险:** 高 QPS 场景下内存增长/扫描成本上升
- **缓解:** 严格 7 天保留 + 追加时裁剪；必要时加“每 key 记录上限”或改为“按分钟桶聚合”（可后续迭代）

