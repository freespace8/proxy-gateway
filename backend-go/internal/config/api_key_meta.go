package config

func sanitizeAPIKeyMeta(meta map[string]APIKeyMeta, apiKeys []string) map[string]APIKeyMeta {
	if len(apiKeys) == 0 || len(meta) == 0 {
		return nil
	}

	cleaned := make(map[string]APIKeyMeta)
	for _, apiKey := range apiKeys {
		m, ok := meta[apiKey]
		if !ok {
			continue
		}
		if !m.Disabled && m.Description == "" {
			continue
		}
		cleaned[apiKey] = m
	}

	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func (u *UpstreamConfig) IsAPIKeyDisabled(apiKey string) bool {
	if u == nil || u.APIKeyMeta == nil {
		return false
	}
	meta, ok := u.APIKeyMeta[apiKey]
	return ok && meta.Disabled
}

// GetEnabledAPIKeys 返回启用状态的 key（默认全部启用）。
func (u *UpstreamConfig) GetEnabledAPIKeys() []string {
	if u == nil {
		return nil
	}
	if u.APIKeyMeta == nil {
		return u.APIKeys
	}
	enabled := make([]string, 0, len(u.APIKeys))
	for _, apiKey := range u.APIKeys {
		if u.IsAPIKeyDisabled(apiKey) {
			continue
		}
		enabled = append(enabled, apiKey)
	}
	return enabled
}
