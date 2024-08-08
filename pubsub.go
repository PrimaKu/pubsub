package pubsub

import (
	"context"
)

type Administer interface {
	CreateTopic(ctx context.Context, topicId string) error
	EnsureTopicExists(ctx context.Context, topicId string) error
	DeleteTopic(ctx context.Context, topicId string) error
	GetTopics(ctx context.Context) ([]string, error)

	CreateSubscription(ctx context.Context, topicId, subscriptionId string, conf *SubscriptionConfig) error
	EnsureSubscriptionExists(ctx context.Context, topicId, subscriptionId string, conf *SubscriptionConfig) error
	DeleteSubscription(ctx context.Context, subscriptionId string) error
	GetSubscriptions(ctx context.Context) ([]string, error)
}

type Publisher interface {
	EnsureTopicExists(ctx context.Context, topicId string) error
	Publish(ctx context.Context, topicId string, msg Message) (string, error)
}

type Subscriber interface {
	EnsureTopicExists(ctx context.Context, topicId string) error
	EnsureSubscriptionExists(ctx context.Context, topicId, subscriptionId string, conf *SubscriptionConfig) error
	Subscribe(ctx context.Context, subscriptionId string, conf SubscribeConfig, handler MessageHandler) error
}

type MessageHandler func(ctx context.Context, msg Message) error
