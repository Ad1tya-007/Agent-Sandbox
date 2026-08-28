package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
	ws "github.com/Ad1tya-007/Agent-Sandbox/backend/internal/websocket"
)

func TestDecodeJSONAcceptsCreateInput(t *testing.T) {
	body := `{
		"name": "research-agent",
		"image": "python:3.12-slim",
		"cpu": "500m",
		"memory": "1Gi",
		"persistentStorage": false
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/sandboxes", strings.NewReader(body))
	var in models.CreateInput
	if err := decodeJSON(req, &in); err != nil {
		t.Fatal(err)
	}
	if in.Name != "research-agent" || in.Image != "python:3.12-slim" || in.CPU != "500m" || in.Memory != "1Gi" || in.PersistentStorage {
		t.Fatalf("decoded = %+v", in)
	}
}

func TestDecodeJSONRejectsMalformed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/sandboxes", strings.NewReader("{"))
	var in models.CreateInput
	err := decodeJSON(req, &in)
	var apiErr *models.Error
	if !models.AsError(err, &apiErr) || apiErr.Kind != models.KindInvalid {
		t.Fatalf("err = %v, want invalid", err)
	}
}

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	payload := `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/sandboxes", strings.NewReader(payload))
	var in models.CreateInput
	err := decodeJSON(req, &in)
	var apiErr *models.Error
	if !models.AsError(err, &apiErr) || apiErr.Kind != models.KindInvalid {
		t.Fatalf("err = %v, want invalid", err)
	}
}

func TestWriteErrorStatusMapping(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		msg    string
	}{
		{"invalid", models.Invalid("bad name"), http.StatusBadRequest, "bad name"},
		{"not_found", models.NotFound(`Sandbox "demo" not found.`), http.StatusNotFound, `Sandbox "demo" not found.`},
		{"conflict", models.Conflict(`Sandbox "demo" already exists.`), http.StatusConflict, `Sandbox "demo" already exists.`},
		{"conflict_state", models.ErrPauseNotRunning, http.StatusConflict, "Only running sandboxes can be paused."},
		{"internal", models.Internal("watch died"), http.StatusInternalServerError, "watch died"},
		{"unknown", errors.New("boom"), http.StatusInternalServerError, "boom"},
		{"nil", nil, http.StatusInternalServerError, "internal error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, tc.err)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
				t.Fatalf("Content-Type = %q", ct)
			}
			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["error"] != tc.msg {
				t.Fatalf("body = %v, want error %q", body, tc.msg)
			}
		})
	}
}

func TestCORSActualRequest(t *testing.T) {
	h := New(nil, ws.New(nil), nil).Handler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "http://127.0.0.1:1420")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:1420" {
		t.Fatalf("Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Fatalf("Allow-Methods = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Content-Type") {
		t.Fatalf("Allow-Headers = %q", got)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	h := New(nil, ws.New(nil), nil).Handler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "http://evil.example")
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty", got)
	}
}

type stubSandboxes struct {
	create func(context.Context, models.CreateInput) (*models.CreateResult, error)
	pause  func(context.Context, string) error
	resume func(context.Context, string) error
	remove func(context.Context, string) error
}

func (s stubSandboxes) Create(ctx context.Context, in models.CreateInput) (*models.CreateResult, error) {
	return s.create(ctx, in)
}
func (s stubSandboxes) Pause(ctx context.Context, name string) error {
	return s.pause(ctx, name)
}
func (s stubSandboxes) Resume(ctx context.Context, name string) error {
	return s.resume(ctx, name)
}
func (s stubSandboxes) Delete(ctx context.Context, name string) error {
	return s.remove(ctx, name)
}

func handlerWith(s Sandboxes) http.Handler {
	return New(s, ws.New(nil), nil).Handler()
}

func TestCreateSuccess(t *testing.T) {
	h := handlerWith(stubSandboxes{
		create: func(_ context.Context, in models.CreateInput) (*models.CreateResult, error) {
			if in.Name != "research-agent" || in.Image != "python:3.12-slim" {
				t.Fatalf("input = %+v", in)
			}
			return &models.CreateResult{Name: in.Name}, nil
		},
	})
	body := `{"name":"research-agent","image":"python:3.12-slim","cpu":"500m","memory":"1Gi","persistentStorage":false}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sandboxes", strings.NewReader(body))
	req.Header.Set("Origin", "http://localhost:1420")
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.Bytes())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:1420" {
		t.Fatalf("Allow-Origin = %q", got)
	}
	var out models.CreateResult
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "research-agent" {
		t.Fatalf("name = %q", out.Name)
	}
}

func TestCreateMalformedJSON(t *testing.T) {
	h := handlerWith(stubSandboxes{
		create: func(context.Context, models.CreateInput) (*models.CreateResult, error) {
			t.Fatal("Create should not be called")
			return nil, nil
		},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sandboxes", strings.NewReader("{"))
	h.ServeHTTP(rec, req)
	assertAPIError(t, rec, http.StatusBadRequest, "malformed JSON")
}

func TestCreateConflict(t *testing.T) {
	h := handlerWith(stubSandboxes{
		create: func(context.Context, models.CreateInput) (*models.CreateResult, error) {
			return nil, models.ErrAlreadyExists.Format("demo")
		},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sandboxes", strings.NewReader(`{"name":"demo"}`))
	h.ServeHTTP(rec, req)
	assertAPIError(t, rec, http.StatusConflict, `Sandbox "demo" already exists.`)
}

func TestPauseResumeDelete(t *testing.T) {
	cases := []struct {
		method string
		path   string
		err    error
		status int
		msg    string
	}{
		{http.MethodPost, "/api/sandboxes/demo/pause", nil, http.StatusNoContent, ""},
		{http.MethodPost, "/api/sandboxes/demo/resume", nil, http.StatusNoContent, ""},
		{http.MethodDelete, "/api/sandboxes/demo", nil, http.StatusNoContent, ""},
		{http.MethodPost, "/api/sandboxes/demo/pause", models.ErrPauseNotRunning, http.StatusConflict, "Only running sandboxes can be paused."},
		{http.MethodPost, "/api/sandboxes/demo/resume", models.ErrResumeNotPaused, http.StatusConflict, "Only paused sandboxes can be resumed."},
		{http.MethodDelete, "/api/sandboxes/missing", models.ErrMissing.Format("missing"), http.StatusNotFound, `Sandbox "missing" not found.`},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			h := handlerWith(stubSandboxes{
				pause:  func(context.Context, string) error { return tc.err },
				resume: func(context.Context, string) error { return tc.err },
				remove: func(context.Context, string) error { return tc.err },
			})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			h.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d body = %s", rec.Code, tc.status, rec.Body.Bytes())
			}
			if tc.msg == "" {
				if rec.Body.Len() != 0 {
					t.Fatalf("204 body = %q", rec.Body.String())
				}
				return
			}
			assertAPIError(t, rec, tc.status, tc.msg)
		})
	}
}

func TestCreateUsesJSONDecoderNotRawBody(t *testing.T) {
	// Ensure camelCase tags round-trip through decodeJSON + writeJSON.
	h := handlerWith(stubSandboxes{
		create: func(_ context.Context, in models.CreateInput) (*models.CreateResult, error) {
			if !in.PersistentStorage {
				t.Fatal("persistentStorage not decoded")
			}
			return &models.CreateResult{Name: in.Name}, nil
		},
	})
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(models.CreateInput{
		Name:              "with-pvc",
		Image:             "busybox",
		CPU:               "100m",
		Memory:            "128Mi",
		PersistentStorage: true,
	}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sandboxes", &buf)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.Bytes())
	}
}

func assertAPIError(t *testing.T, rec *httptest.ResponseRecorder, status int, msg string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d body = %s", rec.Code, status, rec.Body.Bytes())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != msg {
		t.Fatalf("error = %q, want %q", body["error"], msg)
	}
}
