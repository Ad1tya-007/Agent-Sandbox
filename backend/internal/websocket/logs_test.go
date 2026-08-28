package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/logs"
)

func TestHandleLogsRequiresName(t *testing.T) {
	h := HandleLogs(logs.New(nil, nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ws/logs", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "Sandbox name is required." {
		t.Fatalf("body = %v", body)
	}
}

func TestHandleLogsSendsEmptySnapshot(t *testing.T) {
	srv := httptest.NewServer(HandleLogs(logs.New(nil, nil)))
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "?name=demo"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != TypeSnapshot {
		t.Fatalf("type = %v", got["type"])
	}
	if _, ok := got["lines"].([]any); !ok {
		t.Fatalf("lines want [], got %s", data)
	}
}
