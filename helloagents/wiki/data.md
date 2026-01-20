# 数据模型

## 概述
项目使用 **文件配置** 作为主要数据来源：
- 运行时配置：`.config/config.json`
- 配置备份：`.config/backups/`（保留最近若干份）

---

## 数据项（摘要）

### Config（概念模型）

**用途:** 保存 Messages/Responses/Gemini 各自的渠道列表、负载策略、以及少量全局开关。

**关键字段（示例级说明）**
| 字段 | 类型 | 说明 |
|------|------|------|
| upstream / responsesUpstream / geminiUpstream | array | 不同 API 类型的渠道配置 |
| loadBalance / responsesLoadBalance / geminiLoadBalance | string | 负载策略（如 failover/round-robin 等） |
| fuzzyModeEnabled | bool | 模糊 failover 策略开关 |

> 具体字段以代码中的 `backend-go/internal/config` 为准。

