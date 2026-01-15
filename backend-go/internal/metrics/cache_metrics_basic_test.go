package metrics

import (
	"math"
	"sync"
	"testing"
)

func TestCacheMetrics_SnapshotAndConcurrent(t *testing.T) {
	var m CacheMetrics

	if got := m.Snapshot(); got != (CacheMetricsSnapshot{}) {
		t.Fatalf("zero Snapshot() = %+v, want all zeros", got)
	}

	m.IncReadHit()
	m.IncReadMiss()
	m.IncWriteSet()
	m.IncWriteUpdate()
	m.SetEntries(123)
	m.SetCapacity(456)

	got := m.Snapshot()
	if got.ReadHit != 1 || got.ReadMiss != 1 || got.WriteSet != 1 || got.WriteUpdate != 1 {
		t.Fatalf("after Inc* Snapshot() = %+v, want each counter=1", got)
	}
	if got.Entries != 123 || got.Capacity != 456 {
		t.Fatalf("after Set* Snapshot() = %+v, want entries=123 capacity=456", got)
	}

	m.SetEntries(math.MaxInt64)
	m.SetCapacity(math.MaxInt64 - 1)
	got = m.Snapshot()
	if got.Entries != math.MaxInt64 || got.Capacity != math.MaxInt64-1 {
		t.Fatalf("large Set* Snapshot() = %+v, want entries/capacity updated", got)
	}

	const goroutines = 32
	const loops = 2000

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				m.IncReadHit()
				m.IncReadMiss()
				m.IncWriteSet()
				m.IncWriteUpdate()
			}
		}()
	}
	wg.Wait()

	got = m.Snapshot()
	wantDelta := int64(goroutines * loops)
	if got.ReadHit != 1+wantDelta || got.ReadMiss != 1+wantDelta || got.WriteSet != 1+wantDelta || got.WriteUpdate != 1+wantDelta {
		t.Fatalf("concurrent Snapshot() = %+v, want each counter increased by %d", got, wantDelta)
	}
}

