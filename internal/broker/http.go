package broker

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"queue-broker/internal/queue"
)

type Handler struct {
	Broker     *Broker
	GetTimeout time.Duration
}

const StatusClientCanceled int = 499

// PUT /queue/<name>
func (h *Handler) Put(response http.ResponseWriter, request *http.Request) {
	queueName := request.URL.Path[len("/queue/"):]
	var msg queue.Message

	if err := json.NewDecoder(request.Body).Decode(&msg); err != nil || msg.Text == "" {
		http.Error(response, "Invalid message format", http.StatusBadRequest)
		return
	}
	if err := h.Broker.Put(queueName, msg); err != nil {
		http.Error(response, err.Error(), http.StatusTooManyRequests)
		return
	}

	response.WriteHeader(http.StatusOK)
}

// GET /queue/<name>?timeout=10
func (h *Handler) Get(response http.ResponseWriter, request *http.Request) {
	name := request.URL.Path[len("/queue/"):]
	timeOut := h.GetTimeout
	if v := request.URL.Query().Get("timeout"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil || secs < 0 {
			http.Error(response, "bad timeout", http.StatusBadRequest)
			return
		}
		timeOut = time.Duration(secs) * time.Second
	}

	ctx, cancel := context.WithTimeout(request.Context(), timeOut)
	defer cancel()

	msg, err := h.Broker.Get(ctx, name)
	if err != nil {
		switch err {
		case context.DeadlineExceeded:
			http.Error(response, "not found", http.StatusNotFound)
		case context.Canceled:
			http.Error(response, "client canceled", StatusClientCanceled)
		default:
			http.Error(response, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	_ = json.NewEncoder(response).Encode(msg)
}
