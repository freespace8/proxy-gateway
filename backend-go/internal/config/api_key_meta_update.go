package config

import "fmt"

func (cm *ConfigManager) SetAPIKeyDisabled(apiType string, upstreamIndex int, keyIndex int, disabled bool) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var upstreams *[]UpstreamConfig
	switch apiType {
	case "messages":
		upstreams = &cm.config.Upstream
	case "responses":
		upstreams = &cm.config.ResponsesUpstream
	case "gemini":
		upstreams = &cm.config.GeminiUpstream
	default:
		return fmt.Errorf("invalid api type: %s", apiType)
	}

	if upstreamIndex < 0 || upstreamIndex >= len(*upstreams) {
		return fmt.Errorf("invalid upstream index: %d", upstreamIndex)
	}

	upstream := &(*upstreams)[upstreamIndex]
	if keyIndex < 0 || keyIndex >= len(upstream.APIKeys) {
		return fmt.Errorf("invalid key index: %d", keyIndex)
	}

	apiKey := upstream.APIKeys[keyIndex]
	meta := APIKeyMeta{}
	if upstream.APIKeyMeta != nil {
		if existing, ok := upstream.APIKeyMeta[apiKey]; ok {
			meta = existing
		}
	}
	meta.Disabled = disabled

	if !meta.Disabled && meta.Description == "" {
		if upstream.APIKeyMeta != nil {
			delete(upstream.APIKeyMeta, apiKey)
			if len(upstream.APIKeyMeta) == 0 {
				upstream.APIKeyMeta = nil
			}
		}
		return cm.saveConfigLocked(cm.config)
	}

	if upstream.APIKeyMeta == nil {
		upstream.APIKeyMeta = make(map[string]APIKeyMeta)
	}
	upstream.APIKeyMeta[apiKey] = meta

	return cm.saveConfigLocked(cm.config)
}
