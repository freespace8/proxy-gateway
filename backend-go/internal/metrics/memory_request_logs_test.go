package metrics

import (
	"testing"
	"time"
)

func TestMemoryRequestLogStore_AddQuery(t *testing.T) {
	store := NewMemoryRequestLogStore(2)
	base := time.Now().Add(-time.Minute).UTC()

	if err := store.AddRequestLog(RequestLogRecord{RequestID: "r1", APIType: "messages", Timestamp: base.Add(1 * time.Second)}); err != nil {
		t.Fatalf("AddRequestLog(r1) err=%v", err)
	}
	if err := store.AddRequestLog(RequestLogRecord{RequestID: "r2", APIType: "messages", Timestamp: base.Add(2 * time.Second)}); err != nil {
		t.Fatalf("AddRequestLog(r2) err=%v", err)
	}
	// 超出容量，r1 被覆盖
	if err := store.AddRequestLog(RequestLogRecord{RequestID: "r3", APIType: "messages", Timestamp: base.Add(3 * time.Second)}); err != nil {
		t.Fatalf("AddRequestLog(r3) err=%v", err)
	}

	logs, total, err := store.QueryRequestLogs("messages", 50, 0)
	if err != nil {
		t.Fatalf("QueryRequestLogs err=%v", err)
	}
	if total != 2 || len(logs) != 2 {
		t.Fatalf("total/len=%d/%d, want 2/2", total, len(logs))
	}
	if logs[0].RequestID != "r3" || logs[1].RequestID != "r2" {
		t.Fatalf("order=%v,%v, want r3,r2", logs[0].RequestID, logs[1].RequestID)
	}
}

func TestMemoryRequestLogStore_RequestCountAndReset(t *testing.T) {
	store := NewMemoryRequestLogStore(2)
	base := time.Now().Add(-time.Minute).UTC()

	add := func(reqID string, keyID string, ts time.Time) {
		t.Helper()
		if err := store.AddRequestLog(RequestLogRecord{
			RequestID:           reqID,
			APIType:             "messages",
			ChannelIndex:        0,
			ChannelName:         "c0",
			KeyMask:             "***",
			KeyID:               keyID,
			Timestamp:           ts,
			DurationMs:          1,
			StatusCode:          200,
			Success:             true,
			Model:               "m",
			InputTokens:         1,
			OutputTokens:        1,
			CacheReadTokens:     0,
			CacheCreationTokens: 0,
		}); err != nil {
			t.Fatalf("AddRequestLog(%s) err=%v", reqID, err)
		}
	}

	// 容量只有 2，但累计计数应持续增长。
	add("r1", "k1", base.Add(1*time.Second))
	add("r2", "k1", base.Add(2*time.Second))
	add("r3", "k1", base.Add(3*time.Second)) // 覆盖 r1

	if got := store.GetKeyRequestCount("messages", 0, "k1"); got != 3 {
		t.Fatalf("GetKeyRequestCount=%d, want %d", got, 3)
	}
	if got := store.GetTotalRequestCount("messages"); got != 3 {
		t.Fatalf("GetTotalRequestCount=%d, want %d", got, 3)
	}

	logs, total, err := store.QueryRequestLogs("messages", 50, 0)
	if err != nil {
		t.Fatalf("QueryRequestLogs err=%v", err)
	}
	if total != 2 || len(logs) != 2 {
		t.Fatalf("total/len=%d/%d, want 2/2", total, len(logs))
	}

	// ResetKey：计数清零 + 历史日志不可查询。
	store.ResetKey("messages", 0, "k1")
	if got := store.GetKeyRequestCount("messages", 0, "k1"); got != 0 {
		t.Fatalf("GetKeyRequestCount(after reset)=%d, want %d", got, 0)
	}
	if got := store.GetTotalRequestCount("messages"); got != 0 {
		t.Fatalf("GetTotalRequestCount(after reset)=%d, want %d", got, 0)
	}
	logs2, total2, err := store.QueryRequestLogs("messages", 50, 0)
	if err != nil {
		t.Fatalf("QueryRequestLogs(after reset) err=%v", err)
	}
	if total2 != 0 || len(logs2) != 0 {
		t.Fatalf("after reset total/len=%d/%d, want 0/0", total2, len(logs2))
	}

	// Reset 后新增请求应重新计数/可查询（时间戳在 resetAt 之后）。
	add("r4", "k1", time.Now().Add(50*time.Millisecond).UTC())
	if got := store.GetKeyRequestCount("messages", 0, "k1"); got != 1 {
		t.Fatalf("GetKeyRequestCount(after new)=%d, want %d", got, 1)
	}

	// 额外覆盖：channel 级 reset 应清空所有 key（含缺失 keyId 的日志）。
	add("r5", "k2", time.Now().Add(60*time.Millisecond).UTC())
	add("r6", "", time.Now().Add(70*time.Millisecond).UTC()) // unknown
	if got := store.GetTotalRequestCount("messages"); got != 3 {
		t.Fatalf("GetTotalRequestCount(before channel reset)=%d, want %d", got, 3)
	}
	store.ResetChannel("messages", 0)
	if got := store.GetTotalRequestCount("messages"); got != 0 {
		t.Fatalf("GetTotalRequestCount(after channel reset)=%d, want %d", got, 0)
	}
}
