package metrics

import (
	"testing"
	"time"
)

func TestMemoryKeyCircuitLogStore_UpsertGetAndTTL(t *testing.T) {
	store := NewMemoryKeyCircuitLogStore(50 * time.Millisecond)

	if err := store.UpsertKeyCircuitLog("messages", "k1", `{"a":1}`); err != nil {
		t.Fatalf("UpsertKeyCircuitLog() err=%v", err)
	}

	got, ok, err := store.GetKeyCircuitLog("messages", "k1")
	if err != nil {
		t.Fatalf("GetKeyCircuitLog() err=%v", err)
	}
	if !ok || got != `{"a":1}` {
		t.Fatalf("GetKeyCircuitLog()=(%q,%v), want (%q,true)", got, ok, `{"a":1}`)
	}

	if err := store.UpsertKeyCircuitLog("messages", "k1", `{"a":2}`); err != nil {
		t.Fatalf("UpsertKeyCircuitLog(update) err=%v", err)
	}
	got, ok, err = store.GetKeyCircuitLog("messages", "k1")
	if err != nil {
		t.Fatalf("GetKeyCircuitLog(update) err=%v", err)
	}
	if !ok || got != `{"a":2}` {
		t.Fatalf("GetKeyCircuitLog(update)=(%q,%v), want (%q,true)", got, ok, `{"a":2}`)
	}

	time.Sleep(70 * time.Millisecond)
	_, ok, err = store.GetKeyCircuitLog("messages", "k1")
	if err != nil {
		t.Fatalf("GetKeyCircuitLog(expired) err=%v", err)
	}
	if ok {
		t.Fatalf("GetKeyCircuitLog(expired) ok=true, want false")
	}
}

