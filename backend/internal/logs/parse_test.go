package logs

import (
	"testing"
	"time"
)

func TestParseLineKubernetesTimestamp(t *testing.T) {
	got := ParseLine("2026-08-27T20:01:02.123456789Z hello world", 3)
	if got.TS != "2026-08-27T20:01:02.123456789Z" {
		t.Fatalf("ts = %q", got.TS)
	}
	if got.Message != "hello world" {
		t.Fatalf("message = %q", got.Message)
	}
	if got.ID == "" {
		t.Fatal("id is empty")
	}
	other := ParseLine("2026-08-27T20:01:02.123456789Z hello world", 4)
	if other.ID == got.ID {
		t.Fatal("offset must change id")
	}
}

func TestParseLineRFC3339(t *testing.T) {
	got := ParseLine("2026-08-27T20:01:02Z started", 0)
	if got.TS != "2026-08-27T20:01:02Z" || got.Message != "started" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseLineWithoutTimestamp(t *testing.T) {
	got := ParseLine("plain log line", 1)
	if got.Message != "plain log line" {
		t.Fatalf("message = %q", got.Message)
	}
	if _, err := time.Parse(time.RFC3339Nano, got.TS); err != nil {
		if _, err2 := time.Parse(time.RFC3339, got.TS); err2 != nil {
			t.Fatalf("ts = %q, not RFC3339", got.TS)
		}
	}
}

func TestParseLineTimestampOnly(t *testing.T) {
	got := ParseLine("2026-08-27T20:01:02.123456789Z", 0)
	if got.TS != "2026-08-27T20:01:02.123456789Z" {
		t.Fatalf("ts = %q", got.TS)
	}
	if got.Message != "" {
		t.Fatalf("message = %q", got.Message)
	}
}

func TestParseLineStableHash(t *testing.T) {
	a := ParseLine("2026-08-27T20:01:02Z hi", 9)
	b := ParseLine("2026-08-27T20:01:02Z hi", 9)
	if a.ID != b.ID {
		t.Fatalf("id %q vs %q", a.ID, b.ID)
	}
}
