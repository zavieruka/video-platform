package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
	"github.com/zavieruka/video-platform/backend/internal/models"
)

type Publisher struct {
	client     *pubsub.Client
	publishers map[string]*pubsub.Publisher
}

func NewPublisher(ctx context.Context, projectID string, topicIDs map[string]string) (*Publisher, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	publishers := make(map[string]*pubsub.Publisher)

	for key, topicID := range topicIDs {
		publisher := client.Publisher(topicID)

		publishers[key] = publisher
	}

	return &Publisher{
		client:     client,
		publishers: publishers,
	}, nil
}

func (p *Publisher) PublishVideoUploaded(ctx context.Context, event *models.VideoUploadedEvent) error {
	publisher, exists := p.publishers["video-uploaded"]
	if !exists {
		return fmt.Errorf("video-uploaded topic not configured")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	result := publisher.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"eventType": "video.uploaded",
			"videoId":   event.VideoID,
		},
	})

	_, err = result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (p *Publisher) PublishProcessingComplete(ctx context.Context, event *models.VideoProcessingCompleteEvent) error {
	publisher, exists := p.publishers["processing-complete"]
	if !exists {
		return fmt.Errorf("processing-complete topic not configured")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	result := publisher.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"eventType": "video.processing.complete",
			"videoId":   event.VideoID,
			"status":    event.Status,
		},
	})

	_, err = result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (p *Publisher) Close() error {
	return p.client.Close()
}
