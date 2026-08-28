package websocket

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		host := u.Hostname()
		return host == "localhost" || host == "127.0.0.1"
	},
}

func (h *Hub) HandleWatch(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	c := newClient()
	writeDone := make(chan struct{})
	go func() {
		h.writePump(conn, c)
		close(writeDone)
	}()
	h.attach(c)

	conn.SetReadLimit(1 << 20)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	h.detach(c)
	<-writeDone
}

func (h *Hub) writePump(conn *websocket.Conn, c *client) {
	defer conn.Close()
	for msg := range c.send {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteJSON(msg); err != nil {
			return
		}
	}
}
