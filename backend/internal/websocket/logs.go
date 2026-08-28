package websocket

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/logs"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

func HandleLogs(mgr *logs.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name == "" {
			writeWSError(w, http.StatusBadRequest, "Sandbox name is required.")
			return
		}
		if mgr == nil {
			writeWSError(w, http.StatusInternalServerError, "log stream is not available")
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		snapshot, lines, stop := mgr.Subscribe(name)
		defer stop()

		if err := conn.WriteJSON(LogSnapshotEvent{
			Type:  TypeSnapshot,
			Lines: models.NonNil(snapshot),
		}); err != nil {
			return
		}

		readErr := make(chan error, 1)
		go func() {
			conn.SetReadLimit(1 << 20)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					readErr <- err
					return
				}
			}
		}()

		for {
			select {
			case <-readErr:
				return
			case line, ok := <-lines:
				if !ok {
					return
				}
				if err := conn.WriteJSON(LogLineEvent{Type: TypeLine, Line: line}); err != nil {
					return
				}
			}
		}
	}
}

func writeWSError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
