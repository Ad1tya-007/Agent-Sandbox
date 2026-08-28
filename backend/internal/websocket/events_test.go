package websocket

import (
	"encoding/json"
	"testing"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

func TestWatchEventJSON(t *testing.T) {
	t.Run("connection error", func(t *testing.T) {
		raw, err := json.Marshal(ConnectionEvent{
			Type: TypeConnection,
			Connection: models.Connection{
				State:   models.ConnectionError,
				Cluster: nil,
				Message: models.Ptr("load kubeconfig"),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got["type"] != TypeConnection {
			t.Fatalf("type = %v", got["type"])
		}
		conn, _ := got["connection"].(map[string]any)
		if conn["state"] != "error" || conn["cluster"] != nil {
			t.Fatalf("connection = %s", raw)
		}
	})

	t.Run("empty snapshot is array", func(t *testing.T) {
		raw, err := json.Marshal(SnapshotEvent{
			Type:      TypeSnapshot,
			Sandboxes: models.NonNil([]models.Sandbox(nil)),
		})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got["type"] != TypeSnapshot {
			t.Fatalf("type = %v", got["type"])
		}
		if _, ok := got["sandboxes"].([]any); !ok {
			t.Fatalf("sandboxes want [], got %s", raw)
		}
	})

	t.Run("sandbox added", func(t *testing.T) {
		raw, err := json.Marshal(SandboxEvent{
			Type: TypeSandboxAdded,
			Sandbox: models.Sandbox{
				Name:       "demo",
				Status:     models.PhasePending,
				Conditions: []models.Condition{},
				Events:     []models.TimelineEvent{},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got["type"] != "sandbox.added" {
			t.Fatalf("type = %v", got["type"])
		}
		sb, _ := got["sandbox"].(map[string]any)
		if sb["name"] != "demo" || sb["status"] != "Pending" {
			t.Fatalf("sandbox = %s", raw)
		}
	})

	t.Run("deleted", func(t *testing.T) {
		raw, err := json.Marshal(DeletedEvent{Type: TypeSandboxDeleted, Name: "demo"})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got["type"] != "sandbox.deleted" || got["name"] != "demo" {
			t.Fatalf("got %s", raw)
		}
	})
}

func TestLogEventJSON(t *testing.T) {
	raw, err := json.Marshal(LogSnapshotEvent{
		Type:  TypeSnapshot,
		Lines: models.NonNil([]models.LogLine(nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["lines"].([]any); !ok {
		t.Fatalf("lines want [], got %s", raw)
	}

	raw, err = json.Marshal(LogLineEvent{
		Type: TypeLine,
		Line: models.LogLine{ID: "1", TS: "2026-08-27T20:01:02Z", Message: "hi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "line" {
		t.Fatalf("type = %v", got["type"])
	}
	line, _ := got["line"].(map[string]any)
	if line["id"] != "1" || line["ts"] != "2026-08-27T20:01:02Z" || line["message"] != "hi" {
		t.Fatalf("line = %s", raw)
	}
}
