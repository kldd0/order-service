package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"test-task/order-service/internal/config"
	nats_streaming "test-task/order-service/internal/nats-streaming"
	"test-task/order-service/internal/storage/postgres"

	"github.com/gorilla/mux"
)

func main() {
	// config init
	config, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.TODO()

	s, err := postgres.New(config.DSN())

	if err != nil {
		log.Fatal("failed connecting to database: ", err)
	}
	defer s.Close()

	if err := s.InitDB(ctx); err != nil {
		log.Fatal("failed initializing storage: ", err)
	}

	_, err = nats_streaming.New(fmt.Sprintf("nats://%s", config.NATSAddr()))
	// err = nats_streaming.Init()

	if err != nil {
		log.Fatal("failed initializing nats store: ", err)
	}

	// create http router
	router := mux.NewRouter()

	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "pong")
	}).Methods("GET")

	srv := &http.Server{
		Addr:    config.HTTPAddr(),
		Handler: router,
	}

	log.Printf("Starting HTTP server on %s", config.HTTPAddr())

	// start http server
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe Error: %v", err)
	}
}
