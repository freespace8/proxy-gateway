# 技术设计: 去除 sqlite 指标持久化（纯内存保留 7 天）

## 技术方案

### 核心技术
- Go + Gin（现有）
- 纯内存指标存储（扩展现有 `metrics.MetricsManager` 的 in-memory 历史逻辑）
- 内存 Key 熔断日志存储（替代 `key_circuit_logs` 表）

### 实现要点
- 将 `MetricsManager` 的历史保留窗口从固定 24h 改为可配置（默认 7 天，可复用现有 `METRICS_RETENTION_DAYS`）。
- 历史查询统一走内存：
  - `duration <= retention`：正常返回
  - `duration > retention`：截断为 retention 并返回 warning（保持接口可用、避免误导）
- Key 熔断日志抽象为接口（Upsert/Get），提供内存实现并在写入/读取时做 TTL 过期处理。
- 完全移除 sqlite 实现与依赖：删除 `metrics.SQLiteStore` 相关代码、启动时聚合 goroutine、以及 `go.mod` 的 sqlite 依赖。

## 架构决策 ADR

### ADR-1: 指标存储从 sqlite 改为纯内存（推荐）
**上下文:** 现有 sqlite 仅用于指标持久化与 key 熔断日志；本地低 QPS 场景下磁盘 I/O 与 sqlite 依赖可能带来额外开销与不稳定因素；允许重启丢失数据。
**决策:** 移除 sqlite，指标与历史数据仅在内存中保留最近 7 天；key 熔断日志改为内存保存并按 7 天 TTL 过期。
**理由:** 方案最简单、改动可控、无外部依赖、性能路径更短；满足“保留现有功能但只需 7 天”。
**替代方案:** 保留 sqlite 但默认关闭（env 控制） → 拒绝原因: 仍保留依赖与潜在磁盘 I/O；用户明确希望尽量完全去掉。
**影响:** 重启后历史清空（已确认可接受）；超大流量场景可能需要进一步做桶聚合/限额（保留扩展点）。

## API 设计
- 不新增/变更路径；仅调整历史接口的“降级/截断”策略：
  - 超过 7 天的请求：返回最近 7 天 + `warning`
  - Key circuit-log：7 天外视为不存在（404）

## 数据模型
- 仅内存：
  - 指标历史：沿用现有 `RequestRecord`（每次请求追加一条，按 retention 裁剪）
  - key 熔断日志：`map[apiType+keyID] {logStr, updatedAt}`（按 retention 过期）

## 安全与性能
- **安全:** 不涉及鉴权/密钥保存机制变更；渠道/Key 仍由 `.config/config.json` 管理（必须保留）。
- **性能:** 去除 sqlite 写入/flush/聚合；历史查询走内存（本地低 QPS 更合适）。若后续出现扫描成本问题，再引入“按分钟桶聚合”。

## 测试与部署
- **测试:** 覆盖历史查询 >24h 与 >7d 截断 warning；覆盖 key 熔断日志 TTL；跑 `cd backend-go && go test ./...`。
- **部署:** 无需迁移；可选清理 `.config/metrics.db`（不再使用）。

