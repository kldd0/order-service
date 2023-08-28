package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"test-task/order-service/internal/config"
	"test-task/order-service/internal/nats-streaming/subscriber"
	"test-task/order-service/internal/schema"
	"test-task/order-service/internal/service"
	"test-task/order-service/internal/storage/postgres"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
)

func main() {
	// config init
	config, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	s, err := postgres.New(config.DSN())

	if err != nil {
		log.Fatal("Error: failed connecting to database: ", err)
	}
	defer s.Close()

	if err := s.InitDB(ctx); err != nil {
		log.Fatal("Error: failed initializing storage: ", err)
	}

	// init nats connection
	nc, err := nats.Connect(fmt.Sprintf("nats://%s", config.NATSAddr()))
	// nc, err := nats.Connect(nats.DefaultURL)

	if err != nil {
		log.Fatal("Error: failed connecting to NATS: ", err)
	}
	defer nc.Flush()
	defer nc.Close()

	cm, err := subscriber.New(nc)

	if err != nil {
		log.Fatal("Error: failed creating consumer: ", err)
	}

	// subscribe for messages
	ch, err := cm.Subscribe()

	if err != nil {
		log.Fatal("Error: subscribe to cluster: ", err)
	}

	// start producer app
	go publisher(nc)

	// create http router
	router := mux.NewRouter()

	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "pong")
	}).Methods("GET")

	router.HandleFunc("/orders/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(mux.Vars(r)["id"])

		fmt.Fprintf(w, "/orders/{id:[0-9]+}: %d\n", id)
	}).Methods("GET")

	srv := &http.Server{
		Addr:    config.HTTPAddr(),
		Handler: router,
	}

	svc := service.New(ctx, s)

	// start business logic
	go svc.Run(ch)

	log.Printf("Starting HTTP server on %s", config.HTTPAddr())

	// start http server
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe error: %v", err)
	}
}

func publisher(nc *nats.Conn) {
	const channel = "order-notification"

	log.Println("pub started")

	data, err := os.ReadFile("data/model.json")
	if err != nil {
		log.Fatal("failed reading file")
	}

	var order schema.Order
	_ = json.Unmarshal(data, &order)

	sc, err := stan.Connect("dev", "order-producer", stan.NatsConn(nc),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Fatalf("NATS Connection lost, reason: %v", reason)
		}))

	if err != nil {
		log.Fatal("publisher failed connecting to cluster")
	}

	for i := 0; i < 100000; i++ {
		uuid := uuid.NewString()
		order.OrderUid = uuid[:19]

		data, err = json.Marshal(order)
		if err != nil {
			log.Println("failed marshal data")
		}

		if err := sc.Publish(channel, data); err != nil {
			log.Fatal(err)
		}

		if (i % 100) == 0 {
			fmt.Println("i", i)
		}
		time.Sleep(time.Millisecond * 1)
	}

	log.Println("pub finished")

	errFlush := nc.Flush()
	if errFlush != nil {
		panic(errFlush)
	}

	errLast := nc.LastError()
	if errLast != nil {
		panic(errLast)
	}
}
