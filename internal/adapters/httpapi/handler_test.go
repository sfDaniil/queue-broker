package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"queue-broker/internal/adapters/memory"
	"queue-broker/internal/app"
)

// mustNoErr fails the test immediately when err is not nil.
func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

func newTestServer() http.Handler {
	repo := memory.NewRepo(0, 0)
	svc := app.NewService(repo)
	return NewHandler(svc, 30*time.Second)
}

func TestPutAndGet(t *testing.T) {
	srv := httptest.NewServer(newTestServer())
	defer srv.Close()

	// PUT using helper
	msg := app.Message{Text: "hi"}
	body, _ := json.Marshal(msg)
	resp := doRequest(t, http.MethodPut, srv.URL+"/queue/test", "application/json", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// GET
	resp, err := http.Get(srv.URL + "/queue/test")
	mustNoErr(t, err)
	defer func() { _ = resp.Body.Close() }()
	var got app.Message
	mustNoErr(t, json.NewDecoder(resp.Body).Decode(&got))
	if got != msg {
		t.Fatalf("expected %v, got %v", msg, got)
	}
}

func TestGetTimeoutReturns404(t *testing.T) {
	srv := httptest.NewServer(newTestServer())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/queue/empty?timeout=1") // 1-second timeout
	mustNoErr(t, err)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 on timeout, got %d", resp.StatusCode)
	}
}
