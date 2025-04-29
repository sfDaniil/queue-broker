package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"queue-broker/internal/app"
)

type service interface {
	app.Producer
	app.Consumer
}

type Handler struct {
	svc        service
	getTimeout time.Duration
}

func NewHandler(svc service, defTimeout time.Duration) *Handler {
	return &Handler{svc: svc, getTimeout: defTimeout}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if !strings.HasPrefix(request.URL.Path, "/queue/") {
		http.NotFound(response, request)
		return
	}
	name := strings.TrimPrefix(request.URL.Path, "/queue/")

	switch request.Method {
	case http.MethodPut:
		h.put(response, request, name)
	case http.MethodGet:
		h.get(response, request, name)
	default:
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) put(response http.ResponseWriter, request *http.Request, name string) {
	var msg app.Message
	if err := json.NewDecoder(request.Body).Decode(&msg); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.svc.Put(request.Context(), name, msg); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (h *Handler) get(response http.ResponseWriter, request *http.Request, name string) {
	timeout := h.getTimeout
	if v := request.URL.Query().Get("timeout"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			timeout = time.Duration(secs) * time.Second
		}
	}

	ctx, cancel := context.WithTimeout(request.Context(), timeout)
	defer cancel()

	msg, err := h.svc.Get(ctx, name)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case context.DeadlineExceeded:
			status = http.StatusNotFound // 404 when timeout waiting
		case context.Canceled:
			status = http.StatusRequestTimeout
		}
		http.Error(response, err.Error(), status)
		return
	}
	_ = json.NewEncoder(response).Encode(msg)
}
