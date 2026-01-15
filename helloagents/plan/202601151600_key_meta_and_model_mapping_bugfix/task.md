# 任务清单: 渠道 Key 管理增强 + 模型重定向编辑修复

目录: `helloagents/plan/202601151600_key_meta_and_model_mapping_bugfix/`

---

## 1. 前端：模型重定向编辑 bug 修复
- [ ] 1.1 在 `frontend/src/components/AddChannelModal.vue` 修复 `v-combobox` 选择值类型导致的渲染异常，并回归“编辑映射不消失”场景（why.md#需求-修复模型重定向编辑-场景-编辑现有映射）

## 2. 后端：Key 元信息与选 Key 过滤
- [ ] 2.1 在 `backend-go/internal/config/config.go` 扩展 `UpstreamConfig/UpstreamUpdate`，加入 `apiKeyMeta` 数据结构与默认启用语义
- [ ] 2.2 在 `backend-go/internal/config/config.go` 与相关调用链实现“disabled key 不参与选 Key”（why.md#需求-Key-启用禁用--描述-场景-为-Key-添加描述并禁用）

## 3. 后端：各 APIType 渠道更新支持 apiKeyMeta
- [ ] 3.1 在 `backend-go/internal/config/config_messages.go` 支持更新/清理 `apiKeyMeta`（与 `apiKeys` 保持一致）
- [ ] 3.2 在 `backend-go/internal/config/config_responses.go` 支持更新/清理 `apiKeyMeta`（与 `apiKeys` 保持一致）
- [ ] 3.3 在 `backend-go/internal/config/config_gemini.go` 支持更新/清理 `apiKeyMeta`（与 `apiKeys` 保持一致）

## 4. 后端：Key 启用/禁用切换接口
- [ ] 4.1 新增 handler：按 `channelId + keyIndex` 更新 `disabled`，支持 messages/responses/gemini（how.md#Key-启用禁用切换）
- [ ] 4.2 在 `backend-go/main.go` 注册路由
- [ ] 4.3 在 `backend-go/internal/handlers` 增加接口测试（成功/参数非法/越界/持久化）

## 5. 前端：Key 管理（描述 + 启用/禁用）在弹窗内编辑
- [ ] 5.1 在 `frontend/src/services/api.ts` 扩展 `Channel` 类型，增加 `apiKeyMeta`
- [ ] 5.2 在 `frontend/src/components/AddChannelModal.vue` 增加 Key 描述编辑与启用开关，并在保存时写回 `apiKeyMeta`（why.md#需求-Key-启用禁用--描述-场景-为-Key-添加描述并禁用）

## 6. 前端：渠道编排 Key 表增强
- [ ] 6.1 在 `frontend/src/services/api.ts` 增加 Key 启用/禁用切换 API 方法
- [ ] 6.2 在 `frontend/src/components/ChannelOrchestration.vue` 增加“描述列”与“状态（开关）列”，并接入切换接口（why.md#需求-渠道编排-Key-表增强-场景-列表中查看描述并快速切换-Key）

## 7. 安全检查
- [ ] 7.1 执行安全检查（输入验证、敏感信息处理、鉴权边界、配置兼容与回退）

## 8. 测试
- [ ] 8.1 `cd backend-go && make test` 回归
- [ ] 8.2 前端手工回归：模型映射编辑、Key 描述/开关持久化、禁用 key 不参与实际请求

---

## 任务状态符号
- `[ ]` 待执行
- `[√]` 已完成
- `[X]` 执行失败
- `[-]` 已跳过
- `[?]` 待确认
