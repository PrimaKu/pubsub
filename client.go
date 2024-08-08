package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
)

type Client struct {
	ctx    context.Context
	client *pubsub.Client
}

func NewClient(ctx context.Context, projectID string) (*Client, error) {
	c, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &Client{
		ctx:    ctx,
		client: c,
	}, nil
}

func (c *Client) CreateTopic(ctx context.Context, topicId string) error {
	topic := c.client.Topic(topicId)

	exists, err := topic.Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		topic.Stop()
		return nil
	}

	_, err = c.client.CreateTopic(ctx, topicId)
	return err
}

func (c *Client) DeleteTopic(ctx context.Context, topicId string) error {
	topic := c.client.Topic(topicId)
	return topic.Delete(ctx)
}

func (c *Client) GetTopics(ctx context.Context) ([]string, error) {
	topics := c.client.Topics(ctx)
	var topicIds []string

	for {
		res, err := topics.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			continue
		}
		topicIds = append(topicIds, res.ID())
	}

	return topicIds, nil
}

func (c *Client) EnsureTopicExists(ctx context.Context, topicId string) error {
	topic := c.client.Topic(topicId)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		topic.Stop()
		return nil
	}

	return c.CreateTopic(ctx, topicId)
}

func (c *Client) Publish(ctx context.Context, topicId string, msg Message) (string, error) {
	topic := c.client.Topic(topicId)
	defer topic.Stop()

	if msg.OrderingKey != "" {
		topic.EnableMessageOrdering = true
	}

	pubSubMsg := pubsub.Message{}
	constructPubSubMsg(&pubSubMsg, msg)

	result := topic.Publish(ctx, &pubSubMsg)

	messageId, err := result.Get(ctx)
	if err != nil {
		return "", err
	}

	return messageId, nil
}

func (c *Client) CreateSubscription(ctx context.Context, topicId, subscriptionId string, conf *SubscriptionConfig) error {
	topic := c.client.Topic(topicId)
	sub := c.client.Subscription(subscriptionId)
	isSubExist, err := sub.Exists(ctx)
	if err != nil {
		return err
	}

	if !isSubExist {
		psSubscriptionConf := pubsub.SubscriptionConfig{}
		constructPsSubscriptionConfig(topic, conf, &psSubscriptionConf)

		_, err = c.client.CreateSubscription(ctx, subscriptionId, psSubscriptionConf)
		return err
	}
	return nil
}

func (c *Client) DeleteSubscription(ctx context.Context, subscriptionId string) error {
	sub := c.client.Subscription(subscriptionId)
	return sub.Delete(ctx)
}

func (c *Client) GetSubscriptions(ctx context.Context) ([]string, error) {
	subs := c.client.Subscriptions(ctx)
	var subIds []string

	for {
		res, err := subs.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			continue
		}
		subIds = append(subIds, res.ID())
	}

	return subIds, nil
}

func (c *Client) EnsureSubscriptionExists(ctx context.Context, topicId, subscriptionId string, conf *SubscriptionConfig) error {
	sub := c.client.Subscription(subscriptionId)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return c.CreateSubscription(ctx, topicId, subscriptionId, conf)
}

func (c *Client) Subscribe(ctx context.Context, subscriptionId string, conf SubscribeConfig, handler MessageHandler) error {
	sub := c.client.Subscription(subscriptionId)

	if conf.MaxOutstandingMessages > 0 {
		sub.ReceiveSettings.MaxOutstandingMessages = conf.MaxOutstandingMessages
	}

	if conf.NumOfGoroutines > 0 {
		sub.ReceiveSettings.NumGoroutines = conf.NumOfGoroutines
	}

	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		handleMessage(ctx, msg, handler)
	})
}

func (c *Client) Close() error {
	return c.client.Close()
}

func constructPsSubscriptionConfig(topic *pubsub.Topic, conf *SubscriptionConfig, psSubscriptionConf *pubsub.SubscriptionConfig) {
	psSubscriptionConf.Topic = topic

	if conf.RetentionPeriod > 0 {
		psSubscriptionConf.RetentionDuration = conf.RetentionPeriod
	} else {
		psSubscriptionConf.RetentionDuration = DefaultRetentionPeriod
	}

	if conf.AckDeadline > 0 {
		psSubscriptionConf.AckDeadline = conf.AckDeadline
	} else {
		psSubscriptionConf.AckDeadline = DefaultAckTimeout
	}

	psSubscriptionConf.EnableMessageOrdering = conf.UseMessageOrdering
	psSubscriptionConf.EnableExactlyOnceDelivery = conf.UseExactlyOnceDelivery

	if conf.DeadLetterPolicy != nil {
		psSubscriptionConf.DeadLetterPolicy = &pubsub.DeadLetterPolicy{
			DeadLetterTopic:     conf.DeadLetterPolicy.DeadLetterTopic,
			MaxDeliveryAttempts: conf.DeadLetterPolicy.MaxDeliveryAttempts,
		}
	}

	if conf.RetryPolicy != nil {
		psSubscriptionConf.RetryPolicy = &pubsub.RetryPolicy{
			MinimumBackoff: conf.RetryPolicy.MinBackoffTime,
			MaximumBackoff: conf.RetryPolicy.MaxBackoffTime,
		}
	}
}

func constructPubSubMsg(pubSubMsg *pubsub.Message, msg Message) {
	pubSubMsg.Data = msg.Payload
	pubSubMsg.Attributes = msg.Attributes

	if msg.OrderingKey != "" {
		pubSubMsg.OrderingKey = msg.OrderingKey
	}

	if msg.Id != "" {
		pubSubMsg.ID = msg.Id
	}

	if !msg.Publish.IsZero() {
		pubSubMsg.PublishTime = msg.Publish
	}

	if msg.DeliveryAttempt != nil {
		pubSubMsg.DeliveryAttempt = msg.DeliveryAttempt
	}
}

func handleMessage(ctx context.Context, msg *pubsub.Message, handler MessageHandler) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in pubsub message processing: %v", r)
			msg.Nack()
		}
	}()

	message := Message{
		Id:              msg.ID,
		Payload:         msg.Data,
		Attributes:      msg.Attributes,
		OrderingKey:     msg.OrderingKey,
		Publish:         msg.PublishTime,
		DeliveryAttempt: msg.DeliveryAttempt,
	}

	err := handler(ctx, message)
	if err != nil {
		msg.Nack()
		return
	}

	msg.Ack()
}
