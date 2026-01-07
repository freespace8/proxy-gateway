package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildKeyCircuitLogJSON_MaxBytesAndJSON(t *testing.T) {
	huge := strings.Repeat("A", 20000)
	logStr := BuildKeyCircuitLogJSON("https://example.com", 429, huge, huge)
	if len([]byte(logStr)) > maxKeyCircuitLogBytes {
		t.Fatalf("log size = %d, want <= %d", len([]byte(logStr)), maxKeyCircuitLogBytes)
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(logStr), &obj); err != nil {
		t.Fatalf("expected valid JSON, got err: %v", err)
	}
}

func TestKeyCircuitLog_UpsertAndGet(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "metrics.db")

	store, err := NewSQLiteStore(&SQLiteStoreConfig{DBPath: dbPath, RetentionDays: 3})
	if err != nil {
		t.Fatalf("NewSQLiteStore() err = %v", err)
	}
	defer store.Close()

	if err := store.UpsertKeyCircuitLog("messages", "k1", `{"a":1}`); err != nil {
		t.Fatalf("UpsertKeyCircuitLog() err = %v", err)
	}
	got, ok, err := store.GetKeyCircuitLog("messages", "k1")
	if err != nil {
		t.Fatalf("GetKeyCircuitLog() err = %v", err)
	}
	if !ok || got != `{"a":1}` {
		t.Fatalf("GetKeyCircuitLog() = (%q,%v), want (%q,true)", got, ok, `{"a":1}`)
	}

	if err := store.UpsertKeyCircuitLog("messages", "k1", `{"a":2}`); err != nil {
		t.Fatalf("UpsertKeyCircuitLog(update) err = %v", err)
	}
	got, ok, err = store.GetKeyCircuitLog("messages", "k1")
	if err != nil {
		t.Fatalf("GetKeyCircuitLog(update) err = %v", err)
	}
	if !ok || got != `{"a":2}` {
		t.Fatalf("GetKeyCircuitLog(update) = (%q,%v), want (%q,true)", got, ok, `{"a":2}`)
	}

	_, ok, err = store.GetKeyCircuitLog("messages", "missing")
	if err != nil {
		t.Fatalf("GetKeyCircuitLog(missing) err = %v", err)
	}
	if ok {
		t.Fatalf("GetKeyCircuitLog(missing) ok = true, want false")
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db file exists: %v", err)
	}
}
