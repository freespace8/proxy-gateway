package config

import (
	"log"
	"strings"
)

func cloneStringMapping(mapping map[string]string) map[string]string {
	if len(mapping) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(mapping))
	for source, target := range mapping {
		cloned[source] = target
	}
	return cloned
}

func normalizeGlobalMapping(mapping map[string]string) map[string]string {
	if len(mapping) == 0 {
		return nil
	}

	normalized := make(map[string]string, len(mapping))
	for source, target := range mapping {
		source = strings.TrimSpace(source)
		target = strings.TrimSpace(target)
		if source == "" || target == "" {
			continue
		}
		normalized[source] = target
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeGlobalReasoningMapping(mapping map[string]string) map[string]string {
	if len(mapping) == 0 {
		return nil
	}

	normalized := make(map[string]string, len(mapping))
	for source, target := range mapping {
		source = strings.ToLower(strings.TrimSpace(source))
		target = strings.ToLower(strings.TrimSpace(target))
		if source == "" || target == "" {
			continue
		}
		normalized[source] = target
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

// GetGlobalModelMapping 获取全局模型重定向映射（深拷贝）
func (cm *ConfigManager) GetGlobalModelMapping() map[string]string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cloneStringMapping(cm.config.GlobalModelMapping)
}

// SetGlobalModelMapping 设置全局模型重定向映射
func (cm *ConfigManager) SetGlobalModelMapping(mapping map[string]string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.config.GlobalModelMapping = normalizeGlobalMapping(mapping)
	if err := cm.saveConfigLocked(cm.config); err != nil {
		return err
	}

	log.Printf("[Config-ModelMapping] 已更新全局模型重定向规则，数量=%d", len(cm.config.GlobalModelMapping))
	return nil
}

// GetGlobalReasoningMapping 获取全局思考重定向映射（深拷贝）
func (cm *ConfigManager) GetGlobalReasoningMapping() map[string]string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cloneStringMapping(cm.config.GlobalReasoningMapping)
}

// SetGlobalReasoningMapping 设置全局思考重定向映射
func (cm *ConfigManager) SetGlobalReasoningMapping(mapping map[string]string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.config.GlobalReasoningMapping = normalizeGlobalReasoningMapping(mapping)
	if err := cm.saveConfigLocked(cm.config); err != nil {
		return err
	}

	log.Printf("[Config-ReasoningMapping] 已更新全局思考重定向规则，数量=%d", len(cm.config.GlobalReasoningMapping))
	return nil
}
