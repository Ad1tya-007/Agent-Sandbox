package api

import (
	"context"
	"net/http"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/logs"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/sandbox"
	ws "github.com/Ad1tya-007/Agent-Sandbox/backend/internal/websocket"
)

type Sandboxes interface {
	Create(ctx context.Context, in models.CreateInput) (*models.CreateResult, error)
	Pause(ctx context.Context, name string) error
	Resume(ctx context.Context, name string) error
	Delete(ctx context.Context, name string) error
}

type Server struct {
	sandboxes Sandboxes
	hub       *ws.Hub
	logs      *logs.Manager
}

func New(sandboxes Sandboxes, hub *ws.Hub, logMgr *logs.Manager) *Server {
	return &Server{sandboxes: sandboxes, hub: hub, logs: logMgr}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /api/sandboxes", s.create)
	mux.HandleFunc("POST /api/sandboxes/{name}/pause", s.pause)
	mux.HandleFunc("POST /api/sandboxes/{name}/resume", s.resume)
	mux.HandleFunc("DELETE /api/sandboxes/{name}", s.remove)
	mux.HandleFunc("GET /ws", s.hub.HandleWatch)
	mux.HandleFunc("GET /ws/logs", ws.HandleLogs(s.logs))
	return cors(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var in models.CreateInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	out, err := s.sandboxes.Create(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) pause(w http.ResponseWriter, r *http.Request) {
	if err := s.sandboxes.Pause(r.Context(), r.PathValue("name")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resume(w http.ResponseWriter, r *http.Request) {
	if err := s.sandboxes.Resume(r.Context(), r.PathValue("name")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) remove(w http.ResponseWriter, r *http.Request) {
	if err := s.sandboxes.Delete(r.Context(), r.PathValue("name")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

var _ Sandboxes = (*sandbox.Service)(nil)
