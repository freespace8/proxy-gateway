package metrics

import (
	"testing"
	"time"
)

func TestSQLiteStore_RequestLogs_AddQueryCleanup(t *testing.T) {
	store, err := NewSQLiteStore(&SQLiteStoreConfig{
		DBPath:        t.TempDir() + "/metrics.db",
		RetentionDays: 7,
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore() err = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	base := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	apiType := "messages"

	if err := store.AddRequestLog(RequestLogRecord{
		RequestID:    "req-1",
		ChannelIndex: 1,
		ChannelName:  "ch-1",
		KeyMask:      "sk-****",
		Timestamp:    base.Add(1 * time.Minute),
		DurationMs:   123,
		StatusCode:   200,
		Success:      true,
		Model:        "gpt-test",
		InputTokens:  10,
		OutputTokens: 5,
		CostCents:    7,
		APIType:      apiType,
	}); err != nil {
		t.Fatalf("AddRequestLog(req-1) err = %v", err)
	}
	if err := store.AddRequestLog(RequestLogRecord{
		RequestID:    "req-2",
		ChannelIndex: 2,
		ChannelName:  "ch-2",
		KeyMask:      "sk-****",
		Timestamp:    base.Add(2 * time.Minute),
		DurationMs:   456,
		StatusCode:   500,
		Success:      false,
		Model:        "gpt-test",
		ErrorMessage: "boom",
		APIType:      apiType,
	}); err != nil {
		t.Fatalf("AddRequestLog(req-2) err = %v", err)
	}
	if err := store.AddRequestLog(RequestLogRecord{
		RequestID:    "req-3",
		ChannelIndex: 3,
		ChannelName:  "ch-3",
		KeyMask:      "sk-****",
		Timestamp:    base.Add(3 * time.Minute),
		DurationMs:   789,
		StatusCode:   429,
		Success:      false,
		Model:        "gpt-test",
		ErrorMessage: "rate limited",
		APIType:      apiType,
	}); err != nil {
		t.Fatalf("AddRequestLog(req-3) err = %v", err)
	}

	logs, total, err := store.QueryRequestLogs(apiType, 2, 0)
	if err != nil {
		t.Fatalf("QueryRequestLogs() err = %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(logs) != 2 {
		t.Fatalf("len(logs) = %d, want 2", len(logs))
	}
	if logs[0].RequestID != "req-3" || logs[1].RequestID != "req-2" {
		t.Fatalf("order mismatch: got [%s, %s], want [req-3, req-2]", logs[0].RequestID, logs[1].RequestID)
	}
	if logs[0].Timestamp.Unix() != base.Add(3*time.Minute).Unix() {
		t.Fatalf("logs[0].Timestamp = %d, want %d", logs[0].Timestamp.Unix(), base.Add(3*time.Minute).Unix())
	}

	logs2, total2, err := store.QueryRequestLogs(apiType, 2, 2)
	if err != nil {
		t.Fatalf("QueryRequestLogs(page2) err = %v", err)
	}
	if total2 != 3 {
		t.Fatalf("total2 = %d, want 3", total2)
	}
	if len(logs2) != 1 || logs2[0].RequestID != "req-1" {
		t.Fatalf("page2 mismatch: %+v", logs2)
	}

	// 直接写入一条超过 24h 的旧日志，验证清理逻辑
	oldTs := time.Now().Add(-25 * time.Hour).Unix()
	_, err = store.db.Exec(`
		INSERT INTO request_logs (
			request_id, channel_index, channel_name, key_mask,
			timestamp, duration_ms, status_code, success, api_type
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "req-old", 0, "old", "sk-****", oldTs, 1, 200, 1, apiType)
	if err != nil {
		t.Fatalf("insert old request_logs err = %v", err)
	}

	deleted, err := store.CleanupOldRequestLogs()
	if err != nil {
		t.Fatalf("CleanupOldRequestLogs() err = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	_, total3, err := store.QueryRequestLogs(apiType, 50, 0)
	if err != nil {
		t.Fatalf("QueryRequestLogs(after cleanup) err = %v", err)
	}
	if total3 != 3 {
		t.Fatalf("total3 = %d, want 3", total3)
	}
}

func TestSQLiteStore_AddRequestLog_Validation(t *testing.T) {
	store, err := NewSQLiteStore(&SQLiteStoreConfig{
		DBPath:        t.TempDir() + "/metrics.db",
		RetentionDays: 7,
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore() err = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.AddRequestLog(RequestLogRecord{APIType: "messages"}); err == nil {
		t.Fatalf("AddRequestLog(empty request_id) err = nil, want error")
	}
	if err := store.AddRequestLog(RequestLogRecord{RequestID: "req-1"}); err == nil {
		t.Fatalf("AddRequestLog(empty api_type) err = nil, want error")
	}
}
