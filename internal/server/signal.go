package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// signal is a WebRTC signaling message (session description or ICE
// candidate) relayed from one peer to another.
type signal struct {
	From string          `json:"from"`
	To   string          `json:"to"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// handleEvents joins the caller to the room and streams signaling events to
// it until the connection closes.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	// Event streams are long-lived; lift the server's write timeout.
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		s.logger.Warn("clear write deadline", "err", err)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	id, events, others := s.room.join()
	defer func() {
		s.room.leave(id)
		s.logger.Info("peer left", "id", id)
	}()
	s.logger.Info("peer joined", "id", id, "others", len(others))

	welcome, _ := json.Marshal(map[string]any{"id": id, "peers": others})
	err := writeEvent(w, event{"welcome", welcome})
	if err == nil {
		err = rc.Flush()
	}

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for err == nil {
		select {
		case <-r.Context().Done():
			return
		case ev := <-events:
			err = writeEvent(w, ev)
		case <-keepalive.C:
			_, err = io.WriteString(w, ": keepalive\n\n")
		}
		if err == nil {
			err = rc.Flush()
		}
	}
}

// handleSignal relays a signaling message to its recipient.
func (s *Server) handleSignal(w http.ResponseWriter, r *http.Request) {
	var msg signal
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&msg); err != nil {
		http.Error(w, "invalid message", http.StatusBadRequest)
		return
	}
	if !s.room.has(msg.From) {
		http.Error(w, "unknown sender", http.StatusForbidden)
		return
	}
	data, _ := json.Marshal(msg)
	if !s.room.send(msg.To, event{"signal", data}) {
		http.Error(w, "unknown recipient", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeEvent(w io.Writer, ev event) error {
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.name, ev.data)
	return err
}
