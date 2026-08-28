package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/kubernetes"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/watcher"
)

func TestHandleWatchSendsErrorWhenClusterUnavailable(t *testing.T) {
	src := &fakeWatch{conn: models.Connection{
		State:   models.ConnectionError,
		Cluster: nil,
		Message: models.Ptr("kubeconfig not found"),
	}}
	hub := New(src)
	defer hub.Close()
	conn, cleanup := dialWatch(t, hub)
	defer cleanup()

	var got ConnectionEvent
	readJSON(t, conn, &got)
	if got.Type != TypeConnection {
		t.Fatalf("type = %q, want %s", got.Type, TypeConnection)
	}
	if got.Connection.State != "error" {
		t.Fatalf("state = %q, want error", got.Connection.State)
	}
	if got.Connection.Cluster != nil {
		t.Fatalf("cluster = %v, want null", got.Connection.Cluster)
	}
	if got.Connection.Message == nil || *got.Connection.Message != "kubeconfig not found" {
		t.Fatalf("message = %v, want kubeconfig not found", got.Connection.Message)
	}
}

func TestHandleWatchSendsConnectingWhenClusterReady(t *testing.T) {
	src := connectingWatch("kind-kind")
	hub := New(src)
	defer hub.Close()
	conn, cleanup := dialWatch(t, hub)
	defer cleanup()

	var got ConnectionEvent
	readJSON(t, conn, &got)
	if got.Connection.State != "connecting" {
		t.Fatalf("state = %q, want connecting", got.Connection.State)
	}
	if got.Connection.Cluster == nil || *got.Connection.Cluster != "kind-kind" {
		t.Fatalf("cluster = %v, want kind-kind", got.Connection.Cluster)
	}
	if got.Connection.Message == nil || *got.Connection.Message != "Connecting to cluster" {
		t.Fatalf("message = %v", got.Connection.Message)
	}
}

func TestLateSubscriberGetsConnectionThenSnapshot(t *testing.T) {
	src := connectingWatch("kind-kind")
	src.setConnected([]models.Sandbox{
		{Name: "alpha", Status: models.PhaseRunning, Conditions: []models.Condition{}, Events: []models.TimelineEvent{}},
		{Name: "beta", Status: models.PhasePending, Conditions: []models.Condition{}, Events: []models.TimelineEvent{}},
	})
	hub := New(src)
	defer hub.Close()
	conn, cleanup := dialWatch(t, hub)
	defer cleanup()

	var connEv ConnectionEvent
	readJSON(t, conn, &connEv)
	if connEv.Type != TypeConnection || connEv.Connection.State != models.ConnectionConnected {
		t.Fatalf("connection = %+v", connEv)
	}
	if connEv.Connection.Message == nil || *connEv.Connection.Message != "Watching sandboxes" {
		t.Fatalf("message = %v", connEv.Connection.Message)
	}

	raw := readMap(t, conn)
	if raw["type"] != TypeSnapshot {
		t.Fatalf("second event type = %v", raw["type"])
	}
	list, _ := raw["sandboxes"].([]any)
	if len(list) != 2 {
		t.Fatalf("sandboxes = %v", raw["sandboxes"])
	}
}

func TestWatchSnapshotThenDiffs(t *testing.T) {
	src := connectingWatch("kind-kind")
	hub := New(src)
	defer hub.Close()
	conn, cleanup := dialWatch(t, hub)
	defer cleanup()

	var connecting ConnectionEvent
	readJSON(t, conn, &connecting)
	if connecting.Connection.State != models.ConnectionConnecting {
		t.Fatalf("state = %s", connecting.Connection.State)
	}

	src.setConnected([]models.Sandbox{
		{Name: "alpha", Status: models.PhasePending, Conditions: []models.Condition{}, Events: []models.TimelineEvent{}},
	})
	src.emit(watcher.Event{Type: watcher.EventSynced})

	var connected ConnectionEvent
	readJSON(t, conn, &connected)
	if connected.Type != TypeConnection || connected.Connection.State != models.ConnectionConnected {
		t.Fatalf("connected = %+v", connected)
	}
	raw := readMap(t, conn)
	if raw["type"] != TypeSnapshot {
		t.Fatalf("snapshot type = %v", raw["type"])
	}

	src.emit(watcher.Event{
		Type:    watcher.EventAdded,
		Sandbox: models.Sandbox{Name: "beta", Status: models.PhasePending, Conditions: []models.Condition{}, Events: []models.TimelineEvent{}},
	})
	added := readMap(t, conn)
	if added["type"] != "sandbox.added" {
		t.Fatalf("added type = %v", added["type"])
	}
	sb, _ := added["sandbox"].(map[string]any)
	if sb["name"] != "beta" {
		t.Fatalf("added sandbox = %v", added["sandbox"])
	}

	src.emit(watcher.Event{
		Type:    watcher.EventUpdated,
		Sandbox: models.Sandbox{Name: "beta", Status: models.PhaseRunning, Conditions: []models.Condition{}, Events: []models.TimelineEvent{}},
	})
	updated := readMap(t, conn)
	if updated["type"] != "sandbox.updated" {
		t.Fatalf("updated type = %v", updated["type"])
	}

	src.emit(watcher.Event{Type: watcher.EventDeleted, Name: "beta"})
	deleted := readMap(t, conn)
	if deleted["type"] != "sandbox.deleted" || deleted["name"] != "beta" {
		t.Fatalf("deleted = %v", deleted)
	}
}

func TestHubWithWatcherMissingKubeconfig(t *testing.T) {
	k8s := kubernetes.New("/definitely/not-a-kubeconfig", "")
	watch := watcher.New(k8s, nil)
	hub := New(watch)
	defer hub.Close()
	watch.Start(context.Background())
	defer watch.Stop()

	conn, cleanup := dialWatch(t, hub)
	defer cleanup()

	var got ConnectionEvent
	readJSON(t, conn, &got)
	if got.Type != TypeConnection {
		t.Fatalf("type = %q", got.Type)
	}
	if got.Connection.State != models.ConnectionError {
		t.Fatalf("state = %q, want error", got.Connection.State)
	}
	if got.Connection.Cluster != nil {
		t.Fatalf("cluster = %v, want null", got.Connection.Cluster)
	}
	if got.Connection.Message == nil || *got.Connection.Message == "" {
		t.Fatal("expected error message")
	}
}

func TestWatchBroadcastsToMultipleClients(t *testing.T) {
	src := connectingWatch("kind-kind")
	src.setConnected(nil)
	hub := New(src)
	defer hub.Close()

	a, cleanupA := dialWatch(t, hub)
	defer cleanupA()
	b, cleanupB := dialWatch(t, hub)
	defer cleanupB()

	drainUntilSnapshot(t, a)
	drainUntilSnapshot(t, b)

	src.emit(watcher.Event{
		Type:    watcher.EventAdded,
		Sandbox: models.Sandbox{Name: "gamma", Status: models.PhasePending, Conditions: []models.Condition{}, Events: []models.TimelineEvent{}},
	})
	for _, conn := range []*websocket.Conn{a, b} {
		got := readMap(t, conn)
		if got["type"] != "sandbox.added" {
			t.Fatalf("type = %v", got["type"])
		}
	}
}

type fakeWatch struct {
	mu        sync.Mutex
	conn      models.Connection
	sandboxes []models.Sandbox
	subs      map[int]chan watcher.Event
	next      int
}

func connectingWatch(cluster string) *fakeWatch {
	return &fakeWatch{
		conn: models.Connection{
			State:   models.ConnectionConnecting,
			Cluster: models.NonEmptyPtr(cluster),
			Message: models.Ptr("Connecting to cluster"),
		},
	}
}

func (f *fakeWatch) Subscribe() (<-chan watcher.Event, func()) {
	ch := make(chan watcher.Event, 32)
	f.mu.Lock()
	if f.subs == nil {
		f.subs = map[int]chan watcher.Event{}
	}
	id := f.next
	f.next++
	f.subs[id] = ch
	f.mu.Unlock()
	return ch, func() {
		f.mu.Lock()
		delete(f.subs, id)
		f.mu.Unlock()
	}
}

func (f *fakeWatch) Snapshot() []models.Sandbox {
	f.mu.Lock()
	defer f.mu.Unlock()
	return models.NonNil(append([]models.Sandbox(nil), f.sandboxes...))
}

func (f *fakeWatch) Connection() models.Connection {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.conn
}

func (f *fakeWatch) setConnected(sandboxes []models.Sandbox) {
	f.mu.Lock()
	f.sandboxes = sandboxes
	f.conn = models.Connection{
		State:   models.ConnectionConnected,
		Cluster: f.conn.Cluster,
		Message: models.Ptr("Watching sandboxes"),
	}
	f.mu.Unlock()
}

func (f *fakeWatch) emit(ev watcher.Event) {
	f.mu.Lock()
	subs := make([]chan watcher.Event, 0, len(f.subs))
	for _, ch := range f.subs {
		subs = append(subs, ch)
	}
	f.mu.Unlock()
	for _, ch := range subs {
		ch <- ev
	}
}

func dialWatch(t *testing.T, hub *Hub) (*websocket.Conn, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(hub.HandleWatch))
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		srv.Close()
		t.Fatal(err)
	}
	return conn, func() {
		_ = conn.Close()
		srv.Close()
	}
}

func readJSON(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(v); err != nil {
		t.Fatal(err)
	}
}

func readMap(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func drainUntilSnapshot(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 4; i++ {
		got := readMap(t, conn)
		if got["type"] == TypeSnapshot {
			return
		}
	}
	t.Fatal("did not receive snapshot")
}
