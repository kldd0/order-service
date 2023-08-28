package nats_streaming

type Subscriber interface {
	Subscribe() (func() error, error)
}
