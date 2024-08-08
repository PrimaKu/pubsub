package pubsub

import "time"

type Message struct {
	Id              string
	OrderingKey     string
	Payload         []byte
	Publish         time.Time
	Attributes      map[string]string
	DeliveryAttempt *int
}
