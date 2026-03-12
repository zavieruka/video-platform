package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/zavieruka/video-platform/backend/internal/config"
	"github.com/zavieruka/video-platform/backend/internal/database"
	"github.com/zavieruka/video-platform/backend/internal/transcoder"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/test-transcoder/main.go <videoId>")
		os.Exit(1)
	}

	videoID := os.Args[1]

	_ = godotenv.Load()

	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := cfg.InitializeGCPClients(ctx); err != nil {
		log.Fatalf("Failed to initialize GCP clients: %v", err)
	}
	defer cfg.Close()

	videoRepo := database.NewFirestoreVideoRepository(cfg.FirestoreClient)

	video, err := videoRepo.GetByID(ctx, videoID)
	if err != nil {
		log.Fatalf("Failed to get video: %v", err)
	}

	fmt.Printf("Video: %s\n", video.Title)
	fmt.Printf("Status: %s\n", video.Status)
	fmt.Printf("Object: %s\n", video.ObjectName)
	fmt.Println()

	transcoderClient, err := transcoder.NewClient(ctx, cfg.GCPProjectID, cfg.TranscoderLocation, cfg.TranscoderTemplateID)
	if err != nil {
		log.Fatalf("Failed to create transcoder client: %v", err)
	}
	defer transcoderClient.Close()

	inputURI := fmt.Sprintf("gs://%s/%s", cfg.SourceBucketName, video.ObjectName)
	outputPrefix := fmt.Sprintf("gs://%s/%s/", cfg.ProcessedBucketName, videoID)

	fmt.Printf("Input:  %s\n", inputURI)
	fmt.Printf("Output: %s\n", outputPrefix)
	fmt.Println()

	fmt.Println("Creating transcoder job...")
	jobName, err := transcoderClient.CreateJob(ctx, inputURI, outputPrefix)
	if err != nil {
		log.Fatalf("Failed to create job: %v", err)
	}

	fmt.Printf("Job created: %s\n", jobName)
	fmt.Println()

	fmt.Println("Monitoring job status...")
	for {
		job, err := transcoderClient.GetJob(ctx, jobName)
		if err != nil {
			log.Fatalf("Failed to get job status: %v", err)
		}

		state := job.GetState().String()
		fmt.Printf("[%s] State: %s\n", time.Now().Format("15:04:05"), state)

		switch job.GetState().String() {
		case "SUCCEEDED":
			fmt.Println()
			fmt.Println("✓ Job completed successfully!")
			fmt.Printf("Output location: %s\n", outputPrefix)
			fmt.Println()
			fmt.Println("Files created:")
			fmt.Println("  - 1080p.mp4")
			fmt.Println("  - 720p.mp4")
			fmt.Println("  - 480p.mp4")
			fmt.Println("  - manifest.m3u8")
			return

		case "FAILED":
			fmt.Println()
			fmt.Printf("✗ Job failed: %s\n", job.GetError())
			os.Exit(1)

		case "PENDING", "RUNNING":
			time.Sleep(10 * time.Second)

		default:
			fmt.Printf("Unknown state: %s\n", state)
			time.Sleep(10 * time.Second)
		}
	}
}
