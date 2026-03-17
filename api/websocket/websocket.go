package websocket

import (
	"fmt"
	"net/http"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/Bharath-MR-007/hawk-eye/internal/probes"
)

var upgrader = ws.Upgrader{
	HandshakeTimeout: 5 * time.Second,
	ReadBufferSize:   1024,
	WriteBufferSize:  4096,
	// Allow all origins (restrict in production)
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler holds the WebSocket handler dependencies.
type Handler struct {
	ProbeManager *probes.ProbeManager
}

// NewHandler creates a new WebSocket Handler.
func NewHandler(pm *probes.ProbeManager) *Handler {
	return &Handler{ProbeManager: pm}
}

// HandleLiveUpdates upgrades the HTTP connection to WebSocket and streams
// all probe results in real-time to the connected browser/client.
//
// Usage: GET /api/v1/ws/live
func (h *Handler) HandleLiveUpdates(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "could not upgrade to WebSocket", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Unique subscriber ID per connection
	subID := fmt.Sprintf("ws-%s-%d", r.RemoteAddr, time.Now().UnixNano())
	ch := h.ProbeManager.Subscribe(subID)
	defer h.ProbeManager.Unsubscribe(subID)

	// Send welcome frame
	_ = conn.WriteJSON(map[string]string{
		"type":    "connected",
		"message": "Hawkeye live probe stream started",
		"sub_id":  subID,
	})

	// Ping ticker to detect disconnected clients
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case result, ok := <-ch:
			if !ok {
				return // subscriber channel closed
			}
			if err := conn.WriteJSON(result); err != nil {
				return // client disconnected
			}

		case <-pingTicker.C:
			if err := conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return // client disconnected
			}

		case <-r.Context().Done():
			return
		}
	}
}

// HandleCheckStream streams results for a specific check name only.
//
// Usage: GET /api/v1/ws/checks/{checkName}
// (filter is done client-side for now; server-side filter can be added)
func (h *Handler) HandleCheckStream(w http.ResponseWriter, r *http.Request) {
	h.HandleLiveUpdates(w, r) // reuse — clients can filter by result.type
}
