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
	"test-task/order-service/internal/cache"
	"test-task/order-service/internal/config"
	"test-task/order-service/internal/domain"
	"test-task/order-service/internal/http-server/handlers/order/get"
	logger "test-task/order-service/internal/http-server/middleware"
	"test-task/order-service/internal/nats-streaming/subscriber"
	"test-task/order-service/internal/service"
	"test-task/order-service/internal/storage/postgres"
	"test-task/order-service/internal/utils"
	"time"

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

	db, err := postgres.New(config.DSN())

	if err != nil {
		log.Fatal("Error: failed connecting to database: ", err)
	}
	defer db.Close()

	if err := db.InitDB(ctx); err != nil {
		log.Fatal("Error: failed initializing storage: ", err)
	}

	// init nats connection
	nc, err := nats.Connect(fmt.Sprintf("nats://%s", config.NATSAddr()))

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

	// main service init
	svc := service.New(ctx, db)

	// start business logic
	go svc.Run(ch)

	// creating cache
	cache := cache.New(200)
	if err := cache.RestoreFromDB(log, ctx, config.DSN()); err != nil {
		log.Print("Error: failed restore cache: ", err)
	}

	// create http router
	router := mux.NewRouter()

	// logger mw
	router.Use(logger.New(log))

	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "pong")
	}).Methods("GET")

	router.HandleFunc("/orders/{order_uid:[a-z0-9]{19}}", get.New(log, db, cache)).Methods("GET")

	srv := &http.Server{
		Addr:         config.HTTPAddr(),
		Handler:      router,
		ReadTimeout:  config.Timeout(),
		WriteTimeout: config.Timeout(),
		IdleTimeout:  time.Second * 30,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// start event publisher app
	go publisher(10341, log, nc)

	stopped := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-sigint
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// saving cache to DB
		if err := cache.EvacuateToDB(log, config.DSN()); err != nil {
			log.Fatal("Error: failed evacuate cache: ", err)
		}
		log.Printf("Cache evacuated successfully, length: [%d]", cache.Len())

		log.Print("Stopping server")
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("HTTP Server Shutdown Error: %v", err)
		}
		close(stopped)
	}()

	log.Printf("Starting HTTP server on: %s", config.HTTPAddr())

	// start http server
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("Error: HTTP server ListenAndServe error: ", err)
	}

	<-stopped
}

func publisher(ordersCount int, log *log.Logger, nc *nats.Conn) {
	const channel = "order-notification"

	log.Print("Publisher started")

	data, err := os.ReadFile("data/model.json")
	if err != nil {
		log.Fatal("Error: failed reading file: ", err)
	}

	var order domain.Order
	_ = json.Unmarshal(data, &order)

	sc, err := stan.Connect("dev", "order-producer", stan.NatsConn(nc),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Fatal("Error: NATS connection lost, reason: ", reason)
		}))

	if err != nil {
		log.Fatal("Error: publisher failed connecting to cluster: ", err)
	}

	fo, err := os.Create("data/uids.txt")
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	for i := 0; i < ordersCount; i++ {
		uid := utils.GenerateUID19v2()
		fo.Write([]byte(uid + "\n"))

		order.OrderUid = uid
		order.DateCreated = time.Now()

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

		time.Sleep(time.Millisecond * 1)
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
