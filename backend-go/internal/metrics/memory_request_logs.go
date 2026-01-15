package metrics

import (
	"errors"
	"sort"
	"sync"
	"time"
)

// MemoryRequestLogStore 在内存中保存最近 N 条请求日志（环形缓冲）。
// 用途：用于管理端查看近期日志（内存环形缓冲）。
type MemoryRequestLogStore struct {
	mu       sync.RWMutex
	capacity int

	buf   []RequestLogRecord
	start int
	size  int
}

func NewMemoryRequestLogStore(capacity int) *MemoryRequestLogStore {
	if capacity <= 0 {
		capacity = 500
	}
	return &MemoryRequestLogStore{
		capacity: capacity,
		buf:      make([]RequestLogRecord, capacity),
	}
}

func (s *MemoryRequestLogStore) AddRequestLog(logRecord RequestLogRecord) error {
	if s == nil {
		return errors.New("nil store")
	}
	if logRecord.RequestID == "" {
		return errors.New("empty request_id")
	}
	if logRecord.APIType == "" {
		return errors.New("empty api_type")
	}
	if logRecord.Timestamp.IsZero() {
		logRecord.Timestamp = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := (s.start + s.size) % s.capacity
	s.buf[idx] = logRecord
	if s.size < s.capacity {
		s.size++
	} else {
		s.start = (s.start + 1) % s.capacity
	}
	return nil
}

func (s *MemoryRequestLogStore) QueryRequestLogs(apiType string, limit, offset int) ([]RequestLogRecord, int64, error) {
	if s == nil {
		return nil, 0, errors.New("nil store")
	}
	if apiType == "" {
		return nil, 0, errors.New("empty apiType")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// 拷贝并按 apiType 过滤
	filtered := make([]RequestLogRecord, 0, s.size)
	for i := 0; i < s.size; i++ {
		idx := (s.start + i) % s.capacity
		rec := s.buf[idx]
		if rec.APIType == apiType {
			filtered = append(filtered, rec)
		}
	}

	// 按 timestamp 倒序（最新在前）
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	total := int64(len(filtered))
	if int64(offset) >= total {
		return []RequestLogRecord{}, total, nil
	}
	end := offset + limit
	if int64(end) > total {
		end = int(total)
	}
	out := make([]RequestLogRecord, 0, end-offset)
	out = append(out, filtered[offset:end]...)
	return out, total, nil
}
