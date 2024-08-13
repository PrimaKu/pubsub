package pubsub

import "time"

const (
	MinAckDeadline     = 10 * time.Second
	MaxAckDeadline     = 600 * time.Second
	DefaultAckDeadline = 30 * time.Second

	MinRetentionPeriod     = 10 * time.Minute
	MaxRetentionPeriod     = 7 * 24 * time.Hour
	DefaultRetentionPeriod = 7 * 24 * time.Hour

	MinExtensionPeriod        = 10 * time.Second
	MaxExtensionPeriod        = 600 * time.Second
	DefaultMinExtensionPeriod = 30 * time.Second
	DefaultMaxExtensionPeriod = 60 * time.Second

	DefaultNumOfGoroutines = 5
)

type DeadLetterPolicy struct {
	DeadLetterTopic     string
	MaxDeliveryAttempts int
}

type RetryPolicy struct {
	MinBackoffTime time.Duration
	MaxBackoffTime time.Duration
}

// SubscriptionConfig interface for configure subscription.
type SubscriptionConfig struct {
	// DeadLetterPolicy specifies the policy for handling messages that fail to be processed.
	DeadLetterPolicy *DeadLetterPolicy

	// RetryPolicy specifies the policy for retrying message processing.
	RetryPolicy *RetryPolicy

	// RetentionPeriod specifies the duration for which unacknowledged messages are retained.
	// Defaults to 7 days. Cannot be longer than 7 days or shorter than 10 minutes.
	RetentionPeriod time.Duration

	// AckDeadline specifies the maximum time after a subscriber receives a message before
	// the subscriber should acknowledge the message. Default is 60s.
	AckDeadline time.Duration

	// UseMessageOrdering enables automatic ordering of messages when the same order key is received.
	UseMessageOrdering bool

	// UseExactlyOnceDelivery enables exactly-once delivery semantics.
	UseExactlyOnceDelivery bool
}

// SubscribeConfig represents the configuration options for a PubSub subscriber.
type SubscribeConfig struct {
	// MinExtensionPeriod is the minimum duration by which to extend the ack deadline at a time.
	MinExtensionPeriod time.Duration

	// MaxExtensionPeriod is the maximum duration by which to extend the ack deadline at a time.
	MaxExtensionPeriod time.Duration

	// MaxOutstandingBytes specifies the maximum size of unprocessed messages (unacknowledged but not yet expired).
	MaxOutstandingBytes int

	// MaxOutstandingMessages specifies the maximum number of unprocessed messages (unacknowledged but not yet expired).
	MaxOutstandingMessages int

	// NumOfGoroutines specifies the number of process to consume the PubSub messages.
	NumOfGoroutines int
}

// PublishConfig represents the configuration options for a PubSub publisher.
type PublishConfig struct {
	// OrderingKey specifies the ordering key to be used for message publishing.
	OrderingKey string // Add ordering key on message
}
