# 后端-Scheduler（backend-go/internal/scheduler）

## 目的
提供多渠道调度：优先级、健康度、Trace 亲和与 failover。

## 模块概述
- **职责:** 选择渠道/Key；记录成功/失败；与 metrics 交互实现熔断与健康判断。
- **状态:** ✅稳定
- **最后更新:** 2026-01-20

## 规范
- 调度决策必须可追踪（日志/原因字段）。
- 不可将 key 明文暴露到日志（使用脱敏）。

## 依赖
- `backend-go/internal/config`
- `backend-go/internal/metrics`
- `backend-go/internal/session`

## 变更历史
- （暂无）

