# 任务清单: 去除 sqlite 指标持久化（纯内存保留 7 天）

目录: `helloagents/plan/202601151221_remove_sqlite_metrics/`

---

## 1. 指标历史改为 7 天纯内存
- [ ] 1.1 在 `backend-go/internal/metrics/channel_metrics.go` 中将历史保留窗口从固定 24h 改为按 `EnvConfig.MetricsRetentionDays` 计算的 retention，验证 why.md#需求:-指标历史（7-天）-场景:-查看全局/渠道/Key-历史（duration=today/24h/7d/30d）
- [ ] 1.2 在 `backend-go/internal/metrics/channel_metrics.go` 中将历史查询逻辑统一改为内存（移除 >24h 走 DB 的分支），并实现 `duration>retention` 截断 + warning，验证 why.md#需求:-指标历史（7-天）-场景:-查看全局/渠道/Key-历史（duration=today/24h/7d/30d）

## 2. Key 熔断日志改为内存 + TTL
- [ ] 2.1 在 `backend-go/internal/metrics/` 中新增 `KeyCircuitLogStore` 接口与内存实现（7 天 TTL），验证 why.md#需求:-Key-熔断日志-场景:-查看某-key-的“最后一次失败日志”
- [ ] 2.2 在 `backend-go/internal/handlers/common/circuit_log.go` 与 `backend-go/internal/handlers/key_circuit_log_handler.go` 中改为依赖 `KeyCircuitLogStore`，并保持无数据时的返回码行为，验证 why.md#需求:-Key-熔断日志-场景:-查看某-key-的“最后一次失败日志”

## 3. 移除 sqlite 实现与依赖
- [ ] 3.1 在 `backend-go/main.go` 中移除 sqlite 初始化与 daily_stats 聚合 goroutine，改为初始化内存 stores，验证 why.md#变更内容
- [ ] 3.2 删除 `backend-go/internal/metrics/sqlite_store.go` 及相关测试，并从 `backend-go/go.mod` 移除 `modernc.org/sqlite`，执行 `cd backend-go && go mod tidy`，验证 why.md#变更内容

## 4. 安全检查
- [ ] 4.1 执行安全检查：确认不触碰渠道/Key 持久化（仍在 `.config/config.json`），不输出明文 key 到日志

## 5. 文档更新
- [ ] 5.1 更新 `ENVIRONMENT.md`/`ARCHITECTURE.md` 中关于 `.config/metrics.db` 与持久化的描述（如存在）

## 6. 测试
- [ ] 6.1 在 `backend-go/internal/metrics/` 中补充/调整单测覆盖“7 天历史 + 截断 warning + TTL”，并跑 `cd backend-go && go test ./...`

---

## 任务状态符号
- `[ ]` 待执行
- `[√]` 已完成
- `[X]` 执行失败
- `[-]` 已跳过
- `[?]` 待确认

