package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"test-task/order-service/internal/config"
	"test-task/order-service/internal/http-server/handlers/order/get"
	logger "test-task/order-service/internal/http-server/middleware"
	"test-task/order-service/internal/nats-streaming/subscriber"
	"test-task/order-service/internal/schema"
	"test-task/order-service/internal/service"
	"test-task/order-service/internal/storage/postgres"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// setup logger [dev] -- debug
	log := log.Default()

	// init config
	config, err := config.New()
	if err != nil {
		log.Fatal("Error: failed initializing config: ", err)
	}

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
	go publisher(log, nc)

	// create http router
	router := mux.NewRouter()

	router.Use(logger.New(log))

	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "pong")
	}).Methods("GET")

	router.HandleFunc("/orders/{id:[0-9]+}", get.New(log, s)).Methods("GET")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:    config.HTTPAddr(),
		Handler: router,
	}

	svc := service.New(ctx, s)

	// start business logic
	go svc.Run(ch)

	go func() {
		// start http server
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("Error: HTTP server ListenAndServe error: ", err)
		}
	}()

	log.Printf("Starting HTTP server on: %s", config.HTTPAddr())

	<-done
	log.Print("Stopping server")

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Error: failed to stop server: ", err)
		return
	}
}

func publisher(log *log.Logger, nc *nats.Conn) {
	const channel = "order-notification"

	log.Print("Publisher started")

	data, err := os.ReadFile("data/model.json")
	if err != nil {
		log.Fatal("Error: failed reading file: ", err)
	}

	var order schema.Order
	_ = json.Unmarshal(data, &order)

	sc, err := stan.Connect("dev", "order-producer", stan.NatsConn(nc),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Fatal("Error: NATS connection lost, reason: ", reason)
		}))

	if err != nil {
		log.Fatal("Error: publisher failed connecting to cluster: ", err)
	}

	for i := 0; i < 10; i++ {
		uuid := uuid.NewString()
		order.OrderUid = uuid[:19]

		data, err = json.Marshal(order)
		if err != nil {
			log.Fatal("Error: failed marshal data: ", err)
		}

		if err := sc.Publish(channel, data); err != nil {
			log.Fatal("Error: failed publish message: ", err)
		}

		if (i % 100) == 0 {
			log.Printf("Messages count statistics: %d", i)
		}
		// time.Sleep(time.Millisecond * 1)
	}

	log.Print("Publisher finished")

	errFlush := nc.Flush()
	if errFlush != nil {
		panic(errFlush)
	}

	errLast := nc.LastError()
	if errLast != nil {
		panic(errLast)
	}
}
