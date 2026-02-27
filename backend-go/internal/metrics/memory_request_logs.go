package metrics

import (
	"errors"
	"sort"
	"sync"
	"time"
)

const unknownKeyID = "unknown"

type requestLogKey struct {
	apiType      string
	channelIndex int
	keyID        string
}

type requestLogChannelKey struct {
	apiType      string
	channelIndex int
}

type requestLogCounter struct {
	total    int64
	baseline int64
	resetAt  time.Time
}

// MemoryRequestLogStore 在内存中保存最近 N 条请求日志（环形缓冲）。
// 用途：用于管理端查看近期日志（内存环形缓冲）。
type MemoryRequestLogStore struct {
	mu       sync.RWMutex
	capacity int

	buf    []RequestLogRecord
	start  int
	size   int
	nextID int64

	counters       map[requestLogKey]*requestLogCounter
	channelResetAt map[requestLogChannelKey]time.Time
}

func NewMemoryRequestLogStore(capacity int) *MemoryRequestLogStore {
	if capacity <= 0 {
		capacity = 500
	}
	return &MemoryRequestLogStore{
		capacity:       capacity,
		buf:            make([]RequestLogRecord, capacity),
		counters:       make(map[requestLogKey]*requestLogCounter),
		channelResetAt: make(map[requestLogChannelKey]time.Time),
	}
}

func normalizeKeyID(keyID string) string {
	if keyID == "" {
		return unknownKeyID
	}
	return keyID
}

func (s *MemoryRequestLogStore) getOrCreateCounterLocked(key requestLogKey) *requestLogCounter {
	if s.counters == nil {
		s.counters = make(map[requestLogKey]*requestLogCounter)
	}
	if c, ok := s.counters[key]; ok {
		return c
	}
	c := &requestLogCounter{}
	s.counters[key] = c
	return c
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

	// 为内存日志生成单调递增 ID（仅用于前端稳定渲染/调试）。
	if logRecord.ID == 0 {
		s.nextID++
		logRecord.ID = s.nextID
	}

	// 记录累计计数（口径：每写入一条请求日志即 +1；不随环形缓冲覆盖而减少）。
	normalizedKeyID := normalizeKeyID(logRecord.KeyID)
	counterKey := requestLogKey{
		apiType:      logRecord.APIType,
		channelIndex: logRecord.ChannelIndex,
		keyID:        normalizedKeyID,
	}
	counter := s.getOrCreateCounterLocked(counterKey)
	counter.total++

	idx := (s.start + s.size) % s.capacity
	s.buf[idx] = logRecord
	if s.size < s.capacity {
		s.size++
	} else {
		s.start = (s.start + 1) % s.capacity
	}
	return nil
}

func (s *MemoryRequestLogStore) shouldFilterRecordLocked(rec RequestLogRecord) bool {
	// channel 级 reset：清空该渠道下所有日志（包括 key 信息缺失的日志）
	if s.channelResetAt != nil {
		if resetAt, ok := s.channelResetAt[requestLogChannelKey{apiType: rec.APIType, channelIndex: rec.ChannelIndex}]; ok {
			if !resetAt.IsZero() && rec.Timestamp.Before(resetAt) {
				return true
			}
		}
	}

	// key 级 reset：清空单 key 的日志（按 keyId）
	keyID := normalizeKeyID(rec.KeyID)
	if s.counters != nil {
		if c, ok := s.counters[requestLogKey{apiType: rec.APIType, channelIndex: rec.ChannelIndex, keyID: keyID}]; ok {
			if !c.resetAt.IsZero() && rec.Timestamp.Before(c.resetAt) {
				return true
			}
		}
	}

	return false
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
			if s.shouldFilterRecordLocked(rec) {
				continue
			}
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

func (s *MemoryRequestLogStore) GetRequestLogDetail(apiType string, id int64) (*RequestLogRecord, bool) {
	if s == nil || apiType == "" || id <= 0 {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := 0; i < s.size; i++ {
		idx := (s.start + i) % s.capacity
		rec := s.buf[idx]
		if rec.APIType != apiType || rec.ID != id {
			continue
		}
		if s.shouldFilterRecordLocked(rec) {
			return nil, false
		}
		out := rec
		return &out, true
	}
	return nil, false
}

// GetKeyRequestCount 返回单 Key 的累计请求数（受 ResetKey/ResetChannel 影响）。
func (s *MemoryRequestLogStore) GetKeyRequestCount(apiType string, channelIndex int, keyID string) int64 {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyID = normalizeKeyID(keyID)
	if s.counters == nil {
		return 0
	}
	c, ok := s.counters[requestLogKey{apiType: apiType, channelIndex: channelIndex, keyID: keyID}]
	if !ok || c == nil {
		return 0
	}
	n := c.total - c.baseline
	if n < 0 {
		return 0
	}
	return n
}

// GetTotalRequestCount 返回指定 apiType 的累计请求数（全渠道汇总，受 ResetKey/ResetChannel 影响）。
func (s *MemoryRequestLogStore) GetTotalRequestCount(apiType string) int64 {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sum int64
	for k, c := range s.counters {
		if k.apiType != apiType || c == nil {
			continue
		}
		n := c.total - c.baseline
		if n > 0 {
			sum += n
		}
	}
	return sum
}

// ResetKey 清空单个 Key 的累计计数与可查询日志（不影响环形缓冲内存占用）。
func (s *MemoryRequestLogStore) ResetKey(apiType string, channelIndex int, keyID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	keyID = normalizeKeyID(keyID)
	now := time.Now()
	key := requestLogKey{apiType: apiType, channelIndex: channelIndex, keyID: keyID}
	c := s.getOrCreateCounterLocked(key)
	c.baseline = c.total
	c.resetAt = now
}

// ResetChannel 清空单个渠道的累计计数与可查询日志（不影响环形缓冲内存占用）。
func (s *MemoryRequestLogStore) ResetChannel(apiType string, channelIndex int) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if s.channelResetAt == nil {
		s.channelResetAt = make(map[requestLogChannelKey]time.Time)
	}
	s.channelResetAt[requestLogChannelKey{apiType: apiType, channelIndex: channelIndex}] = now

	for k, c := range s.counters {
		if k.apiType != apiType || k.channelIndex != channelIndex || c == nil {
			continue
		}
		c.baseline = c.total
		c.resetAt = now
	}
}
