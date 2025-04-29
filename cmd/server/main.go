package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"queue-broker/internal/adapters/httpapi"
	"queue-broker/internal/adapters/memory"
	"queue-broker/internal/app"
)

func main() {
	port := flag.String("port", "8080", "listen port")
	maxQ := flag.Int("max-queues", 0, "limit queues")
	maxM := flag.Int("max-messages", 0, "limit messages per queue")
	getTimeOut := flag.Duration("timeout", 30*time.Second, "GET timeout")
	flag.Parse()

	repo := memory.NewRepo(*maxQ, *maxM)
	service := app.NewService(repo)
	handler := httpapi.NewHandler(service, *getTimeOut)

	log.Printf("* http://localhost:%s", *port)
	err := http.ListenAndServe(":"+*port, handler)
	log.Fatal(err)
}
