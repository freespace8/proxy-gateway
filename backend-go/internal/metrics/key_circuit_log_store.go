package metrics

import (
	"errors"
	"sync"
	"time"
)

// KeyCircuitLogStore 保存“某 key 的最后一次失败日志”（按 apiType + keyID 覆盖更新）。
// 仅用于管理端查询展示，不参与转发链路。
type KeyCircuitLogStore interface {
	UpsertKeyCircuitLog(apiType, keyID, logStr string) error
	GetKeyCircuitLog(apiType, keyID string) (string, bool, error)
}

type keyCircuitLogEntry struct {
	logStr    string
	updatedAt time.Time
}

// MemoryKeyCircuitLogStore 纯内存 Key 熔断日志存储（TTL 过期）。
type MemoryKeyCircuitLogStore struct {
	mu  sync.RWMutex
	ttl time.Duration
	m   map[string]keyCircuitLogEntry
}

func NewMemoryKeyCircuitLogStore(ttl time.Duration) *MemoryKeyCircuitLogStore {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	return &MemoryKeyCircuitLogStore{
		ttl: ttl,
		m:   make(map[string]keyCircuitLogEntry),
	}
}

func (s *MemoryKeyCircuitLogStore) UpsertKeyCircuitLog(apiType, keyID, logStr string) error {
	if s == nil {
		return errors.New("nil store")
	}
	if apiType == "" {
		return errors.New("empty apiType")
	}
	if keyID == "" {
		return errors.New("empty keyID")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(time.Now())
	s.m[apiType+"|"+keyID] = keyCircuitLogEntry{logStr: logStr, updatedAt: time.Now()}
	return nil
}

func (s *MemoryKeyCircuitLogStore) GetKeyCircuitLog(apiType, keyID string) (string, bool, error) {
	if s == nil {
		return "", false, errors.New("nil store")
	}
	if apiType == "" {
		return "", false, errors.New("empty apiType")
	}
	if keyID == "" {
		return "", false, errors.New("empty keyID")
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	e, ok := s.m[apiType+"|"+keyID]
	if !ok {
		return "", false, nil
	}
	if s.ttl > 0 && now.Sub(e.updatedAt) > s.ttl {
		delete(s.m, apiType+"|"+keyID)
		return "", false, nil
	}
	return e.logStr, true, nil
}

func (s *MemoryKeyCircuitLogStore) cleanupLocked(now time.Time) {
	if s.ttl <= 0 {
		return
	}
	cutoff := now.Add(-s.ttl)
	for k, v := range s.m {
		if v.updatedAt.Before(cutoff) {
			delete(s.m, k)
		}
	}
}
