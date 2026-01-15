# 技术设计: 渠道 Key 管理增强 + 模型重定向编辑修复

## 技术方案

### 核心技术
- 后端：Go + Gin；配置文件 `.config/config.json`
- 前端：Vue 3 + Vite + Vuetify

### 实现要点

1. **模型重定向编辑 bug 修复（前端）**
   - 将模型候选项统一为 `string[]`，避免 `v-combobox` 选择项回填为对象导致模板中 `.trim()` 报错而引发渲染异常。
   - 编辑映射时，源模型候选过滤逻辑允许当前正在编辑的 source（避免被“已配置模型过滤”误伤）。

2. **Key 元信息（后端配置）**
   - `UpstreamConfig` 新增 `apiKeyMeta`：以“原始 key 字符串”为索引的元信息（`disabled` + `description`）。
   - 兼容策略：`apiKeyMeta` 缺省或 key 不存在元信息时，视为 `disabled=false`（默认启用）。
   - Key 选择逻辑（`GetNextAPIKey` 及相关调用链）过滤掉 disabled 的 key。

3. **Key 快捷开关接口（后端 API）**
   - 新增按 `channelId + keyIndex` 的 Key 元信息更新接口，支持从“渠道编排”表中直接切换启用/禁用。
   - 接口仅更新 `disabled`（描述仍在“编辑渠道弹窗”内通过更新渠道接口维护）。

4. **前端 UI**
   - `AddChannelModal`：Key 列表新增“启用开关”和“描述输入”（描述仅在弹窗内编辑）。
   - `ChannelOrchestration`：Key 表新增“描述列”与“状态（启用/禁用开关）列”；开关调用新接口并做乐观更新。

## API设计

### 渠道列表/仪表盘（字段增强）
- 现有：`GET /api/messages/channels/dashboard?type=messages|responses`
- 现有：`GET /api/gemini/channels`
- 变更：channels 元素新增
  - `apiKeyMeta: { [apiKey: string]: { disabled?: boolean; description?: string } }`

### Key 启用/禁用切换
- `PATCH /api/{apiType}/channels/{id}/keys/index/{keyIndex}/meta`
  - `apiType`: `messages|responses|gemini`
  - 请求：
    - `{"disabled": true|false}`
  - 响应：
    - `{"success": true}`

## 数据模型

### Config.json（增量）
```json
{
  "upstream": [
    {
      "apiKeys": ["k1", "k2"],
      "apiKeyMeta": {
        "k1": { "disabled": false, "description": "主用" },
        "k2": { "disabled": true, "description": "备用" }
      }
    }
  ]
}
```

## 安全与性能

- **安全**
  - 保持管理 API 仍受 `x-api-key` 鉴权（现有机制）。
  - 不在日志中输出原始 key；前端仍按现有方式掩码展示。
- **性能**
  - `apiKeyMeta` 为小型 map，配置读取/写入增量可忽略。
  - Key 过滤在选 Key 处一次性完成，复杂度 O(n)（n=key 数）。

## 测试与部署

- **测试**
  - Go：新增单测覆盖“disabled key 不参与选 Key”、Key meta 更新接口。
  - 前端：手工回归（编辑模型映射不消失、Key 开关与描述展示/持久化）。
- **部署**
  - 配置文件向后兼容；不需要手动迁移。
