package server

import (
	"crypto/rand"
	"encoding/json"
	"sync"
)

// event is a server-sent event delivered to a single peer.
type event struct {
	name string
	data []byte
}

// room is the single meeting. It tracks connected peers and relays
// signaling messages between them; media flows peer-to-peer.
type room struct {
	mu    sync.Mutex
	peers map[string]chan event
}

func newRoom() *room {
	return &room{peers: make(map[string]chan event)}
}

// join adds a new peer and returns its id, its event channel and the ids of
// the peers already in the room.
func (r *room) join() (string, <-chan event, []string) {
	id := rand.Text()
	ch := make(chan event, 256)

	r.mu.Lock()
	defer r.mu.Unlock()
	others := make([]string, 0, len(r.peers))
	for other := range r.peers {
		others = append(others, other)
	}
	r.peers[id] = ch
	return id, ch, others
}

// leave removes a peer and tells everyone else it is gone.
func (r *room) leave(id string) {
	data, _ := json.Marshal(map[string]string{"id": id})

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.peers, id)
	for _, ch := range r.peers {
		select {
		case ch <- event{"leave", data}:
		default:
		}
	}
}

func (r *room) has(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.peers[id]
	return ok
}

// send queues an event for one peer. It reports false if the peer is not in
// the room or is not keeping up.
func (r *room) send(to string, ev event) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch, ok := r.peers[to]
	if !ok {
		return false
	}
	select {
	case ch <- ev:
		return true
	default:
		return false
	}
}
