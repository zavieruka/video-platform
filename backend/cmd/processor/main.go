package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"cloud.google.com/go/pubsub/v2"
	"github.com/joho/godotenv"
	"github.com/zavieruka/video-platform/backend/internal/config"
	"github.com/zavieruka/video-platform/backend/internal/database"
	"github.com/zavieruka/video-platform/backend/internal/models"
	pubsubpkg "github.com/zavieruka/video-platform/backend/internal/pubsub"
	"github.com/zavieruka/video-platform/backend/internal/services"
	"github.com/zavieruka/video-platform/backend/internal/storage"
	"github.com/zavieruka/video-platform/backend/internal/transcoder"
)

func main() {
	log.Println("===========================================")
	log.Println("Video Platform - Processor Service")
	log.Println("===========================================")
	log.Println()

	_ = godotenv.Load()

	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Environment: %s", cfg.Environment)
	log.Printf("Project ID: %s", cfg.GCPProjectID)
	log.Printf("Region: %s", cfg.GCPRegion)
	log.Println()

	if err := cfg.InitializeGCPClients(ctx); err != nil {
		log.Fatalf("Failed to initialize GCP clients: %v", err)
	}
	defer cfg.Close()

	log.Println("GCP clients initialized successfully")

	videoRepo := database.NewFirestoreVideoRepository(cfg.FirestoreClient)
	videoStorage := storage.NewGCSVideoStorage(cfg.StorageClient, cfg.SourceBucketName, cfg.ServiceAccountEmail)

	transcoderClient, err := transcoder.NewClient(ctx, cfg.GCPProjectID, cfg.TranscoderLocation, cfg.TranscoderTemplateID)
	if err != nil {
		log.Fatalf("Failed to create transcoder client: %v", err)
	}
	defer transcoderClient.Close()

	log.Println("Transcoder client initialized successfully")

	processingService := services.NewProcessingService(
		videoRepo,
		videoStorage,
		transcoderClient,
		cfg.SourceBucketName,
		cfg.ProcessedBucketName,
	)

	subscriber, err := pubsubpkg.NewSubscriber(ctx, cfg.GCPProjectID, "video-processor-sub")
	if err != nil {
		log.Fatalf("Failed to create subscriber: %v", err)
	}
	defer subscriber.Close()

	log.Println("Pub/Sub subscriber initialized successfully")
	log.Println()
	log.Println("Listening for video upload events...")
	log.Println("Press Ctrl+C to stop")
	log.Println()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := subscriber.Listen(ctx, func(ctx context.Context, msg *pubsub.Message) error {
			var event models.VideoUploadedEvent
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				return err
			}

			log.Printf("Received event for video %s", event.VideoID)

			if err := processingService.ProcessVideo(ctx, &event); err != nil {
				log.Printf("Failed to process video %s: %v", event.VideoID, err)
				return err
			}

			return nil
		}); err != nil {
			log.Fatalf("Failed to listen: %v", err)
		}
	}()

	<-sigChan
	log.Println()
	log.Println("Shutting down gracefully...")
}
