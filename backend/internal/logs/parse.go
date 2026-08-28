package logs

import (
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

func ParseLine(raw string, offset int) models.LogLine {
	raw = strings.TrimRight(raw, "\r")
	ts, msg := splitTimestamp(raw)
	return models.LogLine{
		ID:      lineID(ts, msg, offset),
		TS:      ts,
		Message: msg,
	}
}

func splitTimestamp(raw string) (ts, msg string) {
	if raw == "" {
		return time.Now().UTC().Format(time.RFC3339Nano), ""
	}
	i := strings.IndexByte(raw, ' ')
	if i <= 0 {
		if isRFC3339(raw) {
			return raw, ""
		}
		return time.Now().UTC().Format(time.RFC3339Nano), raw
	}
	candidate := raw[:i]
	if isRFC3339(candidate) {
		return candidate, raw[i+1:]
	}
	return time.Now().UTC().Format(time.RFC3339Nano), raw
}

func isRFC3339(s string) bool {
	if _, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return true
	}
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

func lineID(ts, msg string, offset int) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(ts))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(msg))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.Itoa(offset)))
	return strconv.FormatUint(h.Sum64(), 16)
}

func parseTime(ts string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t, true
	}
	return time.Time{}, false
}
