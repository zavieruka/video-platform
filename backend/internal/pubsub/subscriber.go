package pubsub

import (
	"context"
	"fmt"

	pubsub "cloud.google.com/go/pubsub/v2"
)

type MessageHandler func(ctx context.Context, msg *pubsub.Message) error

type Subscriber struct {
	client         *pubsub.Client
	projectID      string
	subscriptionID string
	sub            *pubsub.Subscriber
}

func NewSubscriber(ctx context.Context, projectID, subscriptionID string) (*Subscriber, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	sub := client.Subscriber(subscriptionID)

	return &Subscriber{
		client:         client,
		projectID:      projectID,
		subscriptionID: subscriptionID,
		sub:            sub,
	}, nil
}

func (s *Subscriber) Listen(ctx context.Context, handler MessageHandler) error {
	err := s.sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		if err := handler(ctx, msg); err != nil {
			msg.Nack()
			return
		}

		msg.Ack()
	})

	if err != nil {
		return fmt.Errorf("subscriber receive error: %w", err)
	}

	return nil
}

func (s *Subscriber) Close() error {
	return s.client.Close()
}
