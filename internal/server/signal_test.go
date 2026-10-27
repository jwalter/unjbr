package server

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stream struct {
	body io.Closer
	r    *bufio.Reader
}

func connect(t *testing.T, url string) *stream {
	t.Helper()
	resp, err := http.Get(url + "/events")
	if err != nil {
		t.Fatal(err)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}
	return &stream{body: resp.Body, r: bufio.NewReader(resp.Body)}
}

// next returns the name and data of the next event, skipping comments.
func (s *stream) next(t *testing.T) (string, map[string]any) {
	t.Helper()
	var name, data string
	for {
		line, err := s.r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		line = strings.TrimSuffix(line, "\n")
		switch {
		case line == "" && name != "":
			var v map[string]any
			if err := json.Unmarshal([]byte(data), &v); err != nil {
				t.Fatal(err)
			}
			return name, v
		case strings.HasPrefix(line, "event: "):
			name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data = strings.TrimPrefix(line, "data: ")
		}
	}
}

func post(t *testing.T, url string, msg signal) int {
	t.Helper()
	body, _ := json.Marshal(msg)
	resp, err := http.Post(url+"/signal", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestSignaling(t *testing.T) {
	srv := httptest.NewServer(New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	defer srv.Close()

	a := connect(t, srv.URL)
	defer a.body.Close()
	name, welcomeA := a.next(t)
	if name != "welcome" || len(welcomeA["peers"].([]any)) != 0 {
		t.Fatalf("first welcome = %s %v", name, welcomeA)
	}
	idA := welcomeA["id"].(string)

	b := connect(t, srv.URL)
	_, welcomeB := b.next(t)
	idB := welcomeB["id"].(string)
	if peers := welcomeB["peers"].([]any); len(peers) != 1 || peers[0] != idA {
		t.Fatalf("second welcome peers = %v, want [%s]", peers, idA)
	}

	offer := json.RawMessage(`{"type":"offer","sdp":"v=0"}`)
	if code := post(t, srv.URL, signal{From: idB, To: idA, Type: "description", Data: offer}); code != http.StatusNoContent {
		t.Fatalf("relay status = %d", code)
	}
	name, got := a.next(t)
	if name != "signal" || got["from"] != idB || got["type"] != "description" {
		t.Fatalf("relayed = %s %v", name, got)
	}

	if code := post(t, srv.URL, signal{From: "nobody", To: idA}); code != http.StatusForbidden {
		t.Errorf("unknown sender status = %d", code)
	}
	if code := post(t, srv.URL, signal{From: idA, To: "nobody"}); code != http.StatusNotFound {
		t.Errorf("unknown recipient status = %d", code)
	}

	b.body.Close()
	name, got = a.next(t)
	if name != "leave" || got["id"] != idB {
		t.Fatalf("after disconnect = %s %v", name, got)
	}
}
