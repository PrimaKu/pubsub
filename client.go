package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"time"
)

type Client struct {
	ctx       context.Context
	projectId string
	client    *pubsub.Client
}

func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	c, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		ctx:       ctx,
		projectId: projectID,
		client:    c,
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
	if !exists {
		return fmt.Errorf("topic %s not found", topicId)
	}

	topic.Stop()
	return nil
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
		fmt.Printf("[pubsub][onMessagePublishErr] error: %v", err)
		return "", err
	}

	fmt.Printf("[pubsub][onMessagePublished] id: %s", messageId)
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
	constructPsSubscribeConfigMsg(sub, &conf)

	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		message := Message{
			Id:              msg.ID,
			Data:            msg.Data,
			Attributes:      msg.Attributes,
			OrderingKey:     msg.OrderingKey,
			Publish:         msg.PublishTime,
			DeliveryAttempt: msg.DeliveryAttempt,
		}
		fmt.Printf("[pubsub][onMessageReceived] %+v", message)

		err := handler(ctx, message)
		if err != nil {
			msg.Nack()
			fmt.Printf("[pubsub][onMessageNacked] id: %s", message.Id)
			return
		}

		msg.Ack()
		fmt.Printf("[pubsub][onMessageAcked] id: %s", message.Id)
	})
}

func (c *Client) Close() error {
	return c.client.Close()
}

func constructPsSubscriptionConfig(topic *pubsub.Topic, conf *SubscriptionConfig, psSubscriptionConf *pubsub.SubscriptionConfig) {
	psSubscriptionConf.Topic = topic

	psSubscriptionConf.RetentionDuration = DefaultRetentionPeriod
	if conf.RetentionPeriod >= MinRetentionPeriod && conf.RetentionPeriod <= MaxRetentionPeriod {
		psSubscriptionConf.RetentionDuration = conf.RetentionPeriod
	}

	psSubscriptionConf.AckDeadline = DefaultAckDeadline
	if conf.AckDeadline >= MinAckDeadline && conf.AckDeadline <= MaxAckDeadline {
		psSubscriptionConf.AckDeadline = conf.AckDeadline
	}

	psSubscriptionConf.RetainAckedMessages = false
	psSubscriptionConf.ExpirationPolicy = time.Duration(0) // to indicate that the subscription should never expire.

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

func constructPsSubscribeConfigMsg(sub *pubsub.Subscription, conf *SubscribeConfig) {
	if conf.MaxOutstandingMessages != 0 {
		sub.ReceiveSettings.MaxOutstandingMessages = conf.MaxOutstandingMessages
	}

	if conf.MaxOutstandingBytes != 0 {
		sub.ReceiveSettings.MaxOutstandingBytes = conf.MaxOutstandingBytes
	}

	sub.ReceiveSettings.NumGoroutines = DefaultNumOfGoroutines
	if conf.NumOfGoroutines > 0 {
		sub.ReceiveSettings.NumGoroutines = conf.NumOfGoroutines
	}

	sub.ReceiveSettings.MinExtensionPeriod = DefaultMinExtensionPeriod
	if conf.MinExtensionPeriod >= MinExtensionPeriod && conf.MinExtensionPeriod <= MaxExtensionPeriod {
		sub.ReceiveSettings.MinExtensionPeriod = conf.MinExtensionPeriod
	}

	sub.ReceiveSettings.MaxExtensionPeriod = DefaultMaxExtensionPeriod
	if conf.MaxExtensionPeriod >= MinExtensionPeriod && conf.MaxExtensionPeriod <= MaxExtensionPeriod {
		sub.ReceiveSettings.MaxExtensionPeriod = conf.MaxExtensionPeriod
	}
}

func constructPubSubMsg(pubSubMsg *pubsub.Message, msg Message) {
	pubSubMsg.Data = msg.Data
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

func (c *Client) constructTopicName(topicName string) string {
	return fmt.Sprintf("projects/%s/topics/%s", c.projectId, topicName)
}

func (c *Client) constructSubscriptionName(subscriptionName string) string {
	return fmt.Sprintf("projects/%s/subscriptions/%s", c.projectId, subscriptionName)
}
