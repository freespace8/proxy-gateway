package config

import (
	"sort"
	"strings"
	"time"
)

// ============== 工具函数 ==============

// deduplicateStrings 去重字符串切片，保持原始顺序
func deduplicateStrings(items []string) []string {
	if len(items) <= 1 {
		return items
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// deduplicateBaseURLs 去重 BaseURLs，忽略末尾 / 和 # 差异
func deduplicateBaseURLs(urls []string) []string {
	if len(urls) <= 1 {
		return urls
	}
	seen := make(map[string]struct{}, len(urls))
	result := make([]string, 0, len(urls))
	for _, url := range urls {
		normalized := strings.TrimRight(url, "/#")
		if _, exists := seen[normalized]; !exists {
			seen[normalized] = struct{}{}
			result = append(result, url)
		}
	}
	return result
}

// validateLoadBalanceStrategy 验证负载均衡策略
func validateLoadBalanceStrategy(strategy string) error {
	// 只接受 failover 策略（round-robin 和 random 已移除）
	// 为兼容旧配置，仍允许旧值但静默忽略
	if strategy != "failover" && strategy != "round-robin" && strategy != "random" {
		return &ConfigError{Message: "无效的负载均衡策略: " + strategy}
	}
	return nil
}

// ConfigError 配置错误
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

// ============== 模型与思考重定向 ==============

type mappingPair struct {
	source string
	target string
}

func redirectByMapping(value string, mapping map[string]string) (string, bool) {
	if len(mapping) == 0 {
		return value, false
	}

	if value == "" {
		return value, false
	}

	// 直接匹配（精确匹配优先）
	if mapped, ok := mapping[value]; ok {
		return mapped, true
	}

	// 模糊匹配：按源值长度从长到短排序，确保最长匹配优先
	pairs := make([]mappingPair, 0, len(mapping))
	for source, target := range mapping {
		if source == "" {
			continue
		}
		pairs = append(pairs, mappingPair{source: source, target: target})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if len(pairs[i].source) != len(pairs[j].source) {
			return len(pairs[i].source) > len(pairs[j].source)
		}
		return pairs[i].source < pairs[j].source
	})

	for _, pair := range pairs {
		if strings.Contains(value, pair.source) || strings.Contains(pair.source, value) {
			return pair.target, true
		}
	}

	return value, false
}

// RedirectModel 模型重定向（仅单渠道映射）
func RedirectModel(model string, upstream *UpstreamConfig) string {
	if upstream == nil {
		return model
	}
	redirected, _ := redirectByMapping(model, upstream.ModelMapping)
	return redirected
}

// RedirectModelWithGlobal 模型重定向（单渠道优先，全局回退）
func RedirectModelWithGlobal(model string, upstream *UpstreamConfig, globalMapping map[string]string) string {
	if upstream != nil {
		if redirected, matched := redirectByMapping(model, upstream.ModelMapping); matched {
			return redirected
		}
	}
	redirected, _ := redirectByMapping(model, globalMapping)
	return redirected
}

// RedirectReasoningEffort 思考等级重定向
func RedirectReasoningEffort(reasoningEffort string, reasoningMapping map[string]string) string {
	redirected, _ := redirectByMapping(reasoningEffort, reasoningMapping)
	return redirected
}

// ============== 渠道状态与优先级辅助函数 ==============

// GetChannelStatus 获取渠道状态（带默认值处理）
func GetChannelStatus(upstream *UpstreamConfig) string {
	if upstream.Status == "" {
		return "active"
	}
	return upstream.Status
}

// GetChannelPriority 获取渠道优先级（带默认值处理）
func GetChannelPriority(upstream *UpstreamConfig, index int) int {
	if upstream.Priority == 0 {
		return index
	}
	return upstream.Priority
}

// IsChannelInPromotion 检查渠道是否处于促销期
func IsChannelInPromotion(upstream *UpstreamConfig) bool {
	if upstream.PromotionUntil == nil {
		return false
	}
	return time.Now().Before(*upstream.PromotionUntil)
}

// ============== UpstreamConfig 方法 ==============

// Clone 深拷贝 UpstreamConfig（用于避免并发修改问题）
// 在多 BaseURL failover 场景下，需要临时修改 BaseURL 字段，
// 使用深拷贝可避免并发请求之间的竞态条件
func (u *UpstreamConfig) Clone() *UpstreamConfig {
	cloned := *u // 浅拷贝

	// 深拷贝切片字段
	if u.BaseURLs != nil {
		cloned.BaseURLs = make([]string, len(u.BaseURLs))
		copy(cloned.BaseURLs, u.BaseURLs)
	}
	if u.APIKeys != nil {
		cloned.APIKeys = make([]string, len(u.APIKeys))
		copy(cloned.APIKeys, u.APIKeys)
	}
	if u.APIKeyMeta != nil {
		cloned.APIKeyMeta = make(map[string]APIKeyMeta, len(u.APIKeyMeta))
		for k, v := range u.APIKeyMeta {
			cloned.APIKeyMeta[k] = v
		}
	}
	if u.ModelMapping != nil {
		cloned.ModelMapping = make(map[string]string, len(u.ModelMapping))
		for k, v := range u.ModelMapping {
			cloned.ModelMapping[k] = v
		}
	}
	if u.PromotionUntil != nil {
		t := *u.PromotionUntil
		cloned.PromotionUntil = &t
	}

	return &cloned
}

// GetEffectiveBaseURL 获取当前应使用的 BaseURL（纯 failover 模式）
// 优先使用 BaseURL 字段（支持调用方临时覆盖），否则从 BaseURLs 数组获取
func (u *UpstreamConfig) GetEffectiveBaseURL() string {
	// 优先使用 BaseURL（可能被调用方临时设置用于指定本次请求的 URL）
	if u.BaseURL != "" {
		return u.BaseURL
	}

	// 回退到 BaseURLs 数组
	if len(u.BaseURLs) > 0 {
		return u.BaseURLs[0]
	}

	return ""
}

// GetAllBaseURLs 获取所有 BaseURL（用于延迟测试）
func (u *UpstreamConfig) GetAllBaseURLs() []string {
	if len(u.BaseURLs) > 0 {
		return u.BaseURLs
	}
	if u.BaseURL != "" {
		return []string{u.BaseURL}
	}
	return nil
}
