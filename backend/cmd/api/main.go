package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/zavieruka/video-platform/backend/internal/config"
	"github.com/zavieruka/video-platform/backend/internal/database"
	"github.com/zavieruka/video-platform/backend/internal/handlers"
	"github.com/zavieruka/video-platform/backend/internal/services"
	"github.com/zavieruka/video-platform/backend/internal/storage"
	"github.com/zavieruka/video-platform/backend/internal/validation"
)

func main() {
	// Load .env file if it exists (for local development)
	// In Cloud Run, environment variables are set directly
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Video Platform API (Environment: %s)", cfg.Environment)

	// Initialize GCP clients
	if err := cfg.InitializeGCPClients(ctx); err != nil {
		log.Fatalf("Failed to initialize GCP clients: %v", err)
	}
	defer func() {
		if err := cfg.Close(); err != nil {
			log.Printf("Error closing GCP clients: %v", err)
		}
	}()

	log.Println("GCP clients initialized successfully")

	// Initialize services
	videoStorage := storage.NewGCSVideoStorage(cfg.StorageClient, cfg.SourceBucketName, cfg.ServiceAccountEmail)
	videoRepository := database.NewFirestoreVideoRepository(cfg.FirestoreClient)
	videoValidator := validation.NewVideoValidator(cfg.MaxUploadSizeMB, cfg.AllowedVideoFormats)
	videoService := services.NewVideoService(videoRepository, videoStorage, videoValidator, cfg.UploadURLExpiryHrs)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(cfg)
	videoHandler := handlers.NewVideoHandler(videoService)

	log.Println("Services and handlers initialized successfully")

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Health check endpoints (for Cloud Run)
	mux.HandleFunc("/health", healthHandler.HandleHealth)
	mux.HandleFunc("/ready", healthHandler.HandleReady)

	// Video endpoints
	mux.HandleFunc("POST /api/v1/videos/upload-url", videoHandler.RequestUploadURL)
	mux.HandleFunc("POST /api/v1/videos/{id}/confirm", videoHandler.ConfirmUpload)
	mux.HandleFunc("POST /api/v1/videos/{id}/fail", videoHandler.FailUpload)
	mux.HandleFunc("GET /api/v1/videos/{id}", videoHandler.GetVideo)
	mux.HandleFunc("GET /api/v1/videos", videoHandler.ListVideos)

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, "Video Platform API - v0.1.0\n")
	})

	server := &http.Server{
		Addr:         cfg.GetAddress(),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
