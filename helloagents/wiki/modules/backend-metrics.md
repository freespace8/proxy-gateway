# 后端-Metrics（backend-go/internal/metrics）

## 目的
维护渠道健康度、统计指标、请求日志与熔断记录。

## 模块概述
- **职责:** 滑动窗口失败率、熔断恢复、分时段统计、Key 级指标与日志存储接口。
- **状态:** ✅稳定
- **最后更新:** 2026-01-20

## 规范
- 指标计算以可解释为先，避免误配置导致内存膨胀（保留窗口/retention 上限）。

## 依赖
- `backend-go/internal/types`
- `backend-go/internal/utils`

## 变更历史
- （暂无）

