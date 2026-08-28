package api

import (
	"encoding/json"
	"net/http"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

const maxBodyBytes = 1 << 20 // 1 MiB

var allowedOrigins = map[string]struct{}{
	"http://localhost:1420": {},
	"http://127.0.0.1:1420": {},
}

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return models.Invalid("malformed JSON")
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		return models.Invalid("malformed JSON")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	apiErr := models.WrapInternal(err)
	if apiErr == nil {
		apiErr = models.Internal("internal error")
	}
	status := http.StatusInternalServerError
	switch apiErr.Kind {
	case models.KindInvalid:
		status = http.StatusBadRequest
	case models.KindNotFound:
		status = http.StatusNotFound
	case models.KindConflict, models.KindConflictState:
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": apiErr.Message})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowedOrigins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
