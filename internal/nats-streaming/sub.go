package nats_streaming

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
)

const (
	clusterID = "dev"
	clientID  = "order-service"
	channel   = "order-notification"
)

type NatsEventStore struct {
	nc *nats.Conn
	// orderSubscription *nats.Subscription
	// orderChan chan
}

func New(natsAddr string) (*NatsEventStore, error) {
	const op = "nats-streaming.sub.New"

	log.Println(natsAddr)

	// Connect to NATS
	nc, err := nats.Connect(natsAddr)

	if err != nil {
		return nil, fmt.Errorf("%s: connecting to nats: %w", op, err)
	}

	return &NatsEventStore{
		nc: nc,
	}, nil
}

func Init() error {
	const op = "nats-streaming.sub.Init"

	// Connect to NATS
	nc, err := nats.Connect("nats://nats:4222")
	if err != nil {
		return fmt.Errorf("%s: connecting to nats: %w", op, err)
	}

	// Connect to cluster
	sc, err := stan.Connect(clusterID, clientID, stan.NatsConn(nc),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Fatalf("NATS Connection lost, reason: %v", reason)
		}))
	if err != nil {
		return fmt.Errorf("%s: connecting to cluster: %w", op, err)
	}

	log.Printf("Connected to %s clusterID: [%s] clientID: [%s]\n", nats.DefaultURL, clusterID, clientID)

	// Subscribe with manual ack mode, and set AckWait to 60 seconds
	aw, _ := time.ParseDuration("60s")
	_, err = sc.Subscribe(channel, func(msg *stan.Msg) {
		msg.Ack() // Manual ACK

		// Processing the message

		// Handle the message
		log.Printf("Subscribed message from clientID - %s for Order: %+v\n", clientID, msg)
	},
		stan.MaxInflight(25),
		stan.SetManualAckMode(),
		stan.AckWait(aw),
	)
	if err != nil {
		return fmt.Errorf("%s: subscribing to a channel: %w", op, err)
	}

	return nil
}
