package realtime

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	clientCtx, cancel := context.WithCancel(context.Background())
	candidate := &client{queue: make(chan queuedSnapshot, 1), cancel: cancel}
	closedCtx := conn.CloseRead(clientCtx)
	if err := h.register(closedCtx, candidate); err != nil {
		cancel()
		_ = conn.Close(websocket.StatusInternalError, "snapshot unavailable")
		return
	}
	defer h.unregister(candidate)
	defer conn.CloseNow()

	for {
		select {
		case <-closedCtx.Done():
			return
		case snapshot := <-candidate.queue:
			writeCtx, writeCancel := context.WithTimeout(closedCtx, writeTimeout)
			err := wsjson.Write(writeCtx, conn, snapshot.view)
			writeCancel()
			if err != nil {
				return
			}
		}
	}
}
