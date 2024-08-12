package pubsub

import "time"

type Message struct {
	Id              string
	OrderingKey     string
	Data            []byte
	Publish         time.Time
	Attributes      map[string]string
	DeliveryAttempt *int
}
