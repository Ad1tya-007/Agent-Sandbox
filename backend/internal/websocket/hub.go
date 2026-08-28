package websocket

import (
	"sync"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/watcher"
)

const clientBuffer = 32

type WatchSource interface {
	Subscribe() (<-chan watcher.Event, func())
	Snapshot() []models.Sandbox
	Connection() models.Connection
}

type Hub struct {
	src WatchSource

	mu       sync.Mutex
	clients  map[*client]struct{}
	stop     chan struct{}
	stopOnce sync.Once
}

type client struct {
	send         chan any
	needSnapshot bool
	closeOnce    sync.Once
}

func New(src WatchSource) *Hub {
	h := &Hub{
		src:     src,
		clients: make(map[*client]struct{}),
		stop:    make(chan struct{}),
	}
	if src != nil {
		events, unsub := src.Subscribe()
		go h.consume(events, unsub)
	}
	return h
}

func (h *Hub) Close() {
	if h == nil {
		return
	}
	h.stopOnce.Do(func() {
		close(h.stop)
	})
}

func (h *Hub) consume(events <-chan watcher.Event, unsub func()) {
	defer unsub()
	for {
		select {
		case <-h.stop:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			h.dispatch(ev)
		}
	}
}

func (h *Hub) dispatch(ev watcher.Event) {
	switch ev.Type {
	case watcher.EventSynced:
		h.broadcast(ConnectionEvent{Type: TypeConnection, Connection: h.currentConnection()})
		h.broadcast(SnapshotEvent{Type: TypeSnapshot, Sandboxes: h.snapshot()})
	case watcher.EventError:
		h.broadcast(ConnectionEvent{Type: TypeConnection, Connection: h.currentConnection()})
	case watcher.EventAdded:
		h.broadcast(SandboxEvent{Type: TypeSandboxAdded, Sandbox: ev.Sandbox})
	case watcher.EventUpdated:
		h.broadcast(SandboxEvent{Type: TypeSandboxUpdated, Sandbox: ev.Sandbox})
	case watcher.EventDeleted:
		h.broadcast(DeletedEvent{Type: TypeSandboxDeleted, Name: ev.Name})
	}
}

func (h *Hub) attach(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conn := h.currentConnection()
	c.send <- ConnectionEvent{Type: TypeConnection, Connection: conn}
	if conn.State == models.ConnectionConnected {
		c.send <- SnapshotEvent{Type: TypeSnapshot, Sandboxes: h.snapshot()}
	}
	h.clients[c] = struct{}{}
}

func (h *Hub) detach(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	c.closeSend()
}

func (h *Hub) broadcast(msg any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		h.enqueue(c, msg)
	}
}

// enqueue never blocks the watcher. If the per-connection buffer is full the
// event is dropped and needSnapshot is set. The next catch-up send is a fresh
// connection + snapshot rather than a holey diff stream.
func (h *Hub) enqueue(c *client, msg any) {
	if c.needSnapshot {
		if !trySend(c.send, ConnectionEvent{Type: TypeConnection, Connection: h.currentConnection()}) {
			return
		}
		if !trySend(c.send, SnapshotEvent{Type: TypeSnapshot, Sandboxes: h.snapshot()}) {
			return
		}
		c.needSnapshot = false
		return
	}
	if !trySend(c.send, msg) {
		c.needSnapshot = true
	}
}

func (h *Hub) currentConnection() models.Connection {
	if h.src == nil {
		return models.Connection{
			State:   models.ConnectionError,
			Message: models.Ptr("watch not configured"),
		}
	}
	return h.src.Connection()
}

func (h *Hub) snapshot() []models.Sandbox {
	if h.src == nil {
		return []models.Sandbox{}
	}
	return models.NonNil(h.src.Snapshot())
}

func newClient() *client {
	return &client{send: make(chan any, clientBuffer)}
}

func (c *client) closeSend() {
	c.closeOnce.Do(func() { close(c.send) })
}

func trySend(ch chan any, msg any) bool {
	select {
	case ch <- msg:
		return true
	default:
		return false
	}
}

var _ WatchSource = (*watcher.Manager)(nil)
