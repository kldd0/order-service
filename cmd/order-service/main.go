package main

import (
	"fmt"
	"log"
	"net/http"
	"test-task/order-service/internal/config"

	"github.com/gorilla/mux"
)

func main() {
	// config initialization
	config, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	// create http router
	router := mux.NewRouter()

	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text; charset=utf-8")
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
