package server

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"log"
)

type Messaging interface {
	CreateChannel(code string)
	Subscribe(code string)
	PublishMessage(message Message)
}

type PubSubMessaging struct {
	client *pubsub.Client
}

func NewPubSubMessaging(projectID string) *PubSubMessaging {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatal("ERROR when creating pubsub client in NewPubSubMessaging -", err)
	}
	return &PubSubMessaging{
		client: client,
	}
}

func (m *PubSubMessaging) CreateChannel(code string) {
	ctx := context.Background()

	// Check if the channel already exists
	channel := m.client.Topic(code)
	ok, err := channel.Exists(ctx)
	if err != nil {
		log.Fatalf("ERROR unable to get channel %s - %s", code, err)
	}

	if ok {
		log.Printf("Messaging channel %s already exists\n", code)
	} else {
		// Create the channel if it doesn't exist
		log.Println("Creating messaging channel", code)
		channel, err = m.client.CreateTopic(ctx, code)
		if err != nil {
			log.Fatalf("ERROR unable to create channel %s - %s", code, err)
		}
	}
}

func (m *PubSubMessaging) Subscribe(code string) {
	ctx := context.Background()

	channel := m.client.Topic(code)
	sub, err := m.client.CreateSubscription(ctx, code, pubsub.SubscriptionConfig{
		Topic:                 channel,
		EnableMessageOrdering: true,
	})
	if err != nil {
		log.Fatalf("ERROR creating a new subscription with channel %s - %s\n", code, err)
	}


}

func (m *PubSubMessaging) PublishMessage(message Message) {
	ctx := context.Background()

	// Publish message to its' codes channel
	channel := m.client.Topic(message.Code)
	channel.Publish(ctx, toPubSubMessage(message))
}

// toPubSubMessage converts a Message to a pubsub.Message.
func toPubSubMessage(message Message) *pubsub.Message {
	data, err := json.Marshal(message)
	if err != nil {
		log.Fatal("ERROR unable to decode Message -", err)
	}
	return &pubsub.Message{Data: data}
}
