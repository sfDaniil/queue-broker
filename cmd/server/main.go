//go:build integration
// +build integration

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"queue-broker/internal/broker"
)

func main() {
	port := flag.String("port", "8080", "listen port")
	maxQ := flag.Int("max-queues", 0, "limit queues")
	maxM := flag.Int("max-messages", 0, "limit messages per queue")
	getTimeOut := flag.Duration("timeout", 30*time.Second, "GET timeout")
	flag.Parse()

	b := broker.New(*maxQ, *maxM)
	h := &broker.Handler{Broker: b, GetTimeout: *getTimeOut}

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

	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: mux,
	}

	// graceful-shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutdown")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	log.Printf("* http://localhost:%s", *port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
