package metrics

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"time"
	"unicode/utf8"
)

const maxKeyCircuitLogBytes = 8 * 1024

type KeyCircuitLog struct {
	Timestamp    time.Time `json:"timestamp"`
	BaseURL      string    `json:"baseUrl"`
	StatusCode   int       `json:"statusCode,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	ResponseBody string    `json:"responseBody,omitempty"`
	Truncated    bool      `json:"truncated,omitempty"`
}

func HashAPIKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])[:16]
}

func BuildKeyCircuitLogJSON(baseURL string, statusCode int, responseBody string, errMsg string) string {
	entry := KeyCircuitLog{
		Timestamp:  time.Now(),
		BaseURL:    baseURL,
		StatusCode: statusCode,
	}

	overheadBytes := func() int {
		b, _ := json.Marshal(entry)
		return len(b)
	}()

	available := maxKeyCircuitLogBytes - overheadBytes
	if available < 0 {
		available = 0
	}

	respBudget := int(float64(available) * 0.8)
	errBudget := available - respBudget

	respTrunc, respDid := truncateMiddleUTF8(responseBody, respBudget)
	errTrunc, errDid := truncateMiddleUTF8(errMsg, errBudget)

	entry.ResponseBody = respTrunc
	entry.ErrorMessage = errTrunc
	entry.Truncated = respDid || errDid

	for i := 0; i < 6; i++ {
		b, _ := json.Marshal(entry)
		if len(b) <= maxKeyCircuitLogBytes {
			return string(b)
		}
		entry.Truncated = true
		respBudget = int(float64(respBudget) * 0.8)
		errBudget = int(float64(errBudget) * 0.8)
		if respBudget < 0 {
			respBudget = 0
		}
		if errBudget < 0 {
			errBudget = 0
		}
		entry.ResponseBody, _ = truncateMiddleUTF8(responseBody, respBudget)
		entry.ErrorMessage, _ = truncateMiddleUTF8(errMsg, errBudget)
	}

	b, _ := json.Marshal(entry)
	// 最后兜底：仍可能超长（转义导致），保证不超过限制，但可能不是合法 JSON。
	s, _ := truncateMiddleUTF8(string(b), maxKeyCircuitLogBytes)
	return s
}

func truncateMiddleUTF8(s string, maxBytes int) (string, bool) {
	if maxBytes <= 0 {
		if s == "" {
			return "", false
		}
		return "", true
	}
	if len(s) <= maxBytes {
		return s, false
	}
	if len([]byte(s)) <= maxBytes {
		return s, false
	}

	const marker = "\n...(中间省略)...\n"
	if maxBytes <= len(marker) {
		out := marker
		if len(out) > maxBytes {
			out = out[:maxBytes]
			for len(out) > 0 && !utf8.ValidString(out) {
				out = out[:len(out)-1]
			}
		}
		return out, true
	}

	keep := maxBytes - len(marker)
	headBytes := keep / 2
	tailBytes := keep - headBytes

	head := safeUTF8PrefixByBytes(s, headBytes)
	tail := safeUTF8SuffixByBytes(s, tailBytes)
	return head + marker + tail, true
}

func safeUTF8PrefixByBytes(s string, n int) string {
	b := []byte(s)
	if n <= 0 {
		return ""
	}
	if len(b) <= n {
		return s
	}
	out := b[:n]
	for len(out) > 0 && !utf8.Valid(out) {
		out = out[:len(out)-1]
	}
	return string(out)
}

func safeUTF8SuffixByBytes(s string, n int) string {
	b := []byte(s)
	if n <= 0 {
		return ""
	}
	if len(b) <= n {
		return s
	}
	start := len(b) - n
	out := b[start:]
	for len(out) > 0 && !utf8.Valid(out) {
		out = out[1:]
	}
	return string(out)
}

func (s *SQLiteStore) UpsertKeyCircuitLog(apiType, keyID, logStr string) error {
	if s == nil || s.db == nil {
		return sql.ErrConnDone
	}
	_, err := s.db.Exec(`
		INSERT INTO key_circuit_logs (api_type, key_id, log, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(api_type, key_id) DO UPDATE SET
			log = excluded.log,
			updated_at = excluded.updated_at
	`, apiType, keyID, logStr, time.Now().Unix())
	return err
}

func (s *SQLiteStore) GetKeyCircuitLog(apiType, keyID string) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, sql.ErrConnDone
	}
	var logStr string
	err := s.db.QueryRow(`
		SELECT log
		FROM key_circuit_logs
		WHERE api_type = ? AND key_id = ?
	`, apiType, keyID).Scan(&logStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	return logStr, true, nil
}
