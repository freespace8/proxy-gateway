package metrics

import (
	"testing"
	"time"
)

func TestSQLiteStore_AggregateDailyStats(t *testing.T) {
	store, err := NewSQLiteStore(&SQLiteStoreConfig{
		DBPath:        t.TempDir() + "/metrics.db",
		RetentionDays: 7,
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore() err = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	loc := time.FixedZone("CST", 8*3600)
	day := time.Date(2025, 12, 25, 0, 0, 0, 0, loc)
	start := day.Unix()
	end := day.AddDate(0, 0, 1).Unix()

	baseURL := "https://example.com"
	apiKey := "sk-test-1234567890"
	metricsKey := generateMetricsKey(baseURL, apiKey)
	keyMask := "sk-test-****"

	// day 内：messages 3 条（2 成功 1 失败），responses 1 条（成功）
	_, err = store.db.Exec(`
		INSERT INTO request_records
		(metrics_key, base_url, key_mask, timestamp, success, input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens, api_type)
		VALUES
			(?, ?, ?, ?, 1, 10, 5, 3, 0, 'messages'),
			(?, ?, ?, ?, 1, 20, 10, 0, 4, 'messages'),
			(?, ?, ?, ?, 0, 0, 0, 0, 0, 'messages'),
			(?, ?, ?, ?, 1, 7, 2, 1, 1, 'responses'),
			(?, ?, ?, ?, 1, 999, 999, 999, 999, 'messages') -- 恰好落在 end，应被排除
	`,
		metricsKey, baseURL, keyMask, start+10,
		metricsKey, baseURL, keyMask, start+20,
		metricsKey, baseURL, keyMask, start+30,
		metricsKey, baseURL, keyMask, start+40,
		metricsKey, baseURL, keyMask, end,
	)
	if err != nil {
		t.Fatalf("insert request_records err = %v", err)
	}

	if err := store.AggregateDailyStats(day); err != nil {
		t.Fatalf("AggregateDailyStats() err = %v", err)
	}

	type row struct {
		date                string
		apiType             string
		metricsKey          string
		totalRequests       int64
		successCount        int64
		failureCount        int64
		inputTokens         int64
		outputTokens        int64
		cacheCreationTokens int64
		cacheReadTokens     int64
	}

	var got []row
	rows, err := store.db.Query(`
		SELECT date, api_type, metrics_key,
		       total_requests, success_count, failure_count,
		       input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens
		FROM daily_stats
		WHERE date = ?
		ORDER BY api_type ASC
	`, day.Format("2006-01-02"))
	if err != nil {
		t.Fatalf("query daily_stats err = %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r row
		if err := rows.Scan(
			&r.date, &r.apiType, &r.metricsKey,
			&r.totalRequests, &r.successCount, &r.failureCount,
			&r.inputTokens, &r.outputTokens, &r.cacheCreationTokens, &r.cacheReadTokens,
		); err != nil {
			t.Fatalf("scan daily_stats err = %v", err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("daily_stats rows = %d, want 2", len(got))
	}

	if got[0].apiType != "messages" || got[0].metricsKey != metricsKey {
		t.Fatalf("messages row mismatch: %+v", got[0])
	}
	if got[0].totalRequests != 3 || got[0].successCount != 2 || got[0].failureCount != 1 {
		t.Fatalf("messages counts = (%d,%d,%d), want (3,2,1)", got[0].totalRequests, got[0].successCount, got[0].failureCount)
	}
	if got[0].inputTokens != 30 || got[0].outputTokens != 15 || got[0].cacheCreationTokens != 3 || got[0].cacheReadTokens != 4 {
		t.Fatalf("messages tokens = (%d,%d,%d,%d), want (30,15,3,4)",
			got[0].inputTokens, got[0].outputTokens, got[0].cacheCreationTokens, got[0].cacheReadTokens)
	}

	if got[1].apiType != "responses" || got[1].metricsKey != metricsKey {
		t.Fatalf("responses row mismatch: %+v", got[1])
	}
	if got[1].totalRequests != 1 || got[1].successCount != 1 || got[1].failureCount != 0 {
		t.Fatalf("responses counts = (%d,%d,%d), want (1,1,0)", got[1].totalRequests, got[1].successCount, got[1].failureCount)
	}
	if got[1].inputTokens != 7 || got[1].outputTokens != 2 || got[1].cacheCreationTokens != 1 || got[1].cacheReadTokens != 1 {
		t.Fatalf("responses tokens = (%d,%d,%d,%d), want (7,2,1,1)",
			got[1].inputTokens, got[1].outputTokens, got[1].cacheCreationTokens, got[1].cacheReadTokens)
	}

	// 幂等：重复聚合不应产生重复行或改变结果
	if err := store.AggregateDailyStats(day); err != nil {
		t.Fatalf("AggregateDailyStats() second err = %v", err)
	}
	var count int64
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM daily_stats WHERE date = ?`, day.Format("2006-01-02")).Scan(&count); err != nil {
		t.Fatalf("count daily_stats err = %v", err)
	}
	if count != 2 {
		t.Fatalf("daily_stats row count after second aggregate = %d, want 2", count)
	}
}
