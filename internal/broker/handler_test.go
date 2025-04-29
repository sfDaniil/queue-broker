package broker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"queue-broker/internal/broker"
	"queue-broker/internal/queue"
)

// newServer helper constructs an httptest.Server with Handler wired to fresh Broker.
func newServer(t *testing.T, getTimeout time.Duration) (*httptest.Server, *broker.Broker) {
	t.Helper()
	b := broker.New(0, 0)
	h := &broker.Handler{Broker: b, GetTimeout: getTimeout}
	mux := http.NewServeMux()
	mux.HandleFunc("/queue/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			h.Put(w, r)
		case http.MethodGet:
			h.Get(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	srv := httptest.NewServer(mux)
	return srv, b
}

// doRequest is a small helper to build and send an HTTP request with arbitrary method.
func doRequest(t *testing.T, method, url, contentType string, body []byte) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	mustNoErr(t, err)
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	mustNoErr(t, err)
	return resp
}

// TestPutBadRequest checks that malformed payload returns 400.
func TestPutBadRequest(t *testing.T) {
	srv, _ := newServer(t, time.Second)
	defer srv.Close()

	resp := doRequest(t, http.MethodPut, srv.URL+"/queue/test", "application/json", []byte(`{"msg": "oops"}`))
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}

// TestPutAndGetSuccess exercises happy path through HTTP layer.
func TestPutAndGetSuccess(t *testing.T) {
	srv, _ := newServer(t, time.Second)
	defer srv.Close()

	body, _ := json.Marshal(queue.Message{Text: "hi"})
	resp := doRequest(t, http.MethodPut, srv.URL+"/queue/chat", "application/json", body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	resp, err := http.Get(srv.URL + "/queue/chat?timeout=1")
	mustNoErr(t, err)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var m queue.Message
	mustNoErr(t, json.NewDecoder(resp.Body).Decode(&m))
	mustEqual(t, m.Text, "hi")
}

// TestGetNotFoundAfterTimeout should yield 404 when message absent.
func TestGetNotFoundAfterTimeout(t *testing.T) {
	srv, _ := newServer(t, 50*time.Millisecond)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/queue/empty?timeout=0")
	mustNoErr(t, err)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
}

// TestClientCancel returns custom 499 status.
func TestClientCancel(t *testing.T) {
	srv, _ := newServer(t, time.Second)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/queue/wait", nil)
	cancel() // immediately cancel

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != broker.StatusClientCanceled {
			t.Fatalf("want 499, got %d", resp.StatusCode)
		}
	} else if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}
