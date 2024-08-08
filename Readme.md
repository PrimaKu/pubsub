# Golang Pub/Sub Library

This is a Golang library that provides a client for interacting with the Google Cloud Pub/Sub service. It offers a simple and convenient way to manage topics, subscriptions, and publish/consume messages.

## Features

- Create, delete, and retrieve topics
- Create, delete, and retrieve subscriptions
- Publish messages to topics
- Subscribe to topics and process incoming messages
- Support for message ordering and exactly-once delivery
- Configurable dead-letter policies and retry policies
- Automatic handling of message acknowledgments and nacks
- 
## Installation

```bash
go get github.com/PrimaKu/pubsub
```

## Usage
Here's an example of how to use the library:

```go
package main

import (
    "context"
    "github.com/PrimaKu/pubsub"
)

func main() {
    ctx := context.Background()
    client, err := pubsub.NewClient(ctx, "your-project-id")
    if err != nil {
        // Handle error
    }

    // Create a topic
    err = client.CreateTopic(ctx, "my-topic")
    if err != nil {
        // Handle error
    }

    // Publish a message
    msg := pubsub.Message{
        Payload: []byte("Hello, Pub/Sub!"),
    }
    _, err = client.Publish(ctx, "my-topic", msg)
    if err != nil {
        // Handle error
    }

    // Subscribe to a topic
    conf := pubsub.SubscribeConfig{
        MaxOutstandingMessages: 10,
        NumOfGoroutines:        5,
    }
    err = client.Subscribe(ctx, "my-subscription", conf, func(ctx context.Context, msg pubsub.Message) error {
        // Process the message
        return nil
    })
    if err != nil {
        // Handle error
    }
}```