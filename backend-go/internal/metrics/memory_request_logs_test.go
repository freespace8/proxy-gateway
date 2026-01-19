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
