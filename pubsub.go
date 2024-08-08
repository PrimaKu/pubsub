package pubsub

import (
	"context"
)

type Administer interface {
	CreateTopic(ctx context.Context, topicId string) error
	JoinTopic(ctx context.Context, topicId string) error
	DeleteTopic(ctx context.Context, topicId string) error
	GetTopics(ctx context.Context) ([]string, error)

	CreateSubscription(ctx context.Context, topicId, subscriptionId string, conf *SubscriptionConfig) error
	JoinSubscription(ctx context.Context, topicId, subscriptionId string, conf *SubscriptionConfig) error
	DeleteSubscription(ctx context.Context, subscriptionId string) error
	GetSubscriptions(ctx context.Context) ([]string, error)
}

type Publisher interface {
	JoinTopic(ctx context.Context, topicId string) error
	Publish(ctx context.Context, topicId string, msg Message) (string, error)
}

type Subscriber interface {
	JoinTopic(ctx context.Context, topicId string) error
	JoinSubscription(ctx context.Context, topicId, subscriptionId string, conf *SubscriptionConfig) error
	Subscribe(ctx context.Context, subscriptionId string, conf SubscribeConfig, handler MessageHandler) error
}

type MessageHandler func(ctx context.Context, msg Message) error
