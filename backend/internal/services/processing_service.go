package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zavieruka/video-platform/backend/internal/database"
	"github.com/zavieruka/video-platform/backend/internal/models"
	"github.com/zavieruka/video-platform/backend/internal/storage"
	"github.com/zavieruka/video-platform/backend/internal/transcoder"
)

type ProcessingService struct {
	videoRepo        database.VideoRepository
	storage          storage.VideoStorage
	transcoderClient *transcoder.Client
	sourceBucket     string
	processedBucket  string
}

func NewProcessingService(
	videoRepo database.VideoRepository,
	storage storage.VideoStorage,
	transcoderClient *transcoder.Client,
	sourceBucket string,
	processedBucket string,
) *ProcessingService {
	return &ProcessingService{
		videoRepo:        videoRepo,
		storage:          storage,
		transcoderClient: transcoderClient,
		sourceBucket:     sourceBucket,
		processedBucket:  processedBucket,
	}
}

func (s *ProcessingService) ProcessVideo(ctx context.Context, event *models.VideoUploadedEvent) error {
	videoID := event.VideoID

	log.Printf("[PROCESSING] Starting processing for video %s", videoID)

	video, err := s.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return fmt.Errorf("failed to get video: %w", err)
	}

	if video.Status != models.StatusUploaded {
		log.Printf("[PROCESSING] Video %s is not in uploaded status (current: %s), skipping", videoID, video.Status)
		return nil
	}

	now := time.Now().UTC()
	if err := s.videoRepo.UpdateProcessingStatus(ctx, videoID, models.StatusProcessing, &now, nil); err != nil {
		return fmt.Errorf("failed to update status to processing: %w", err)
	}

	inputURI := fmt.Sprintf("gs://%s/%s", s.sourceBucket, video.ObjectName)
	outputURI := fmt.Sprintf("gs://%s/%s/", s.processedBucket, videoID)

	log.Printf("[PROCESSING] Input: %s", inputURI)
	log.Printf("[PROCESSING] Output: %s", outputURI)

	jobName, err := s.transcoderClient.CreateJob(ctx, inputURI, outputURI)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to create transcoder job: %v", err)
		if updateErr := s.videoRepo.UpdateStatus(ctx, videoID, models.StatusFailed, &errorMsg); updateErr != nil {
			log.Printf("[PROCESSING] Failed to update status to failed: %v", updateErr)
		}
		return fmt.Errorf("failed to create transcoder job: %w", err)
	}

	jobID := jobName
	if err := s.videoRepo.UpdateProcessingJobID(ctx, videoID, jobID); err != nil {
		log.Printf("[PROCESSING] Failed to store job ID: %v", err)
	}

	log.Printf("[PROCESSING] Job created: %s", jobName)

	go s.monitorJob(context.Background(), videoID, jobName)

	return nil
}

func (s *ProcessingService) monitorJob(ctx context.Context, videoID, jobName string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	log.Printf("[MONITORING] Started monitoring job for video %s", videoID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[MONITORING] Context cancelled for video %s", videoID)
			return
		case <-ticker.C:
			job, err := s.transcoderClient.GetJob(ctx, jobName)
			if err != nil {
				log.Printf("[MONITORING] Failed to get job status for video %s: %v", videoID, err)
				continue
			}

			state := job.GetState().String()
			log.Printf("[MONITORING] Video %s - Job state: %s", videoID, state)

			switch state {
			case "SUCCEEDED":
				s.handleJobSuccess(ctx, videoID)
				return
			case "FAILED":
				s.handleJobFailure(ctx, videoID, job.GetError().String())
				return
			case "PENDING", "RUNNING":
				continue
			default:
				log.Printf("[MONITORING] Unknown state for video %s: %s", videoID, state)
			}
		}
	}
}

func (s *ProcessingService) handleJobSuccess(ctx context.Context, videoID string) {
	log.Printf("[MONITORING] Job succeeded for video %s", videoID)

	outputPrefix := fmt.Sprintf("%s/", videoID)
	processedVideos := map[string]models.ProcessedVideo{
		"1080p": {
			Resolution: "1080p",
			StorageURL: fmt.Sprintf("gs://%s/%svideo-1080p/", s.processedBucket, outputPrefix),
			PublicURL:  fmt.Sprintf("https://storage.googleapis.com/%s/%svideo-1080p/media.m3u8", s.processedBucket, outputPrefix),
			Bitrate:    5000000,
		},
		"720p": {
			Resolution: "720p",
			StorageURL: fmt.Sprintf("gs://%s/%svideo-720p/", s.processedBucket, outputPrefix),
			PublicURL:  fmt.Sprintf("https://storage.googleapis.com/%s/%svideo-720p/media.m3u8", s.processedBucket, outputPrefix),
			Bitrate:    2500000,
		},
		"480p": {
			Resolution: "480p",
			StorageURL: fmt.Sprintf("gs://%s/%svideo-480p/", s.processedBucket, outputPrefix),
			PublicURL:  fmt.Sprintf("https://storage.googleapis.com/%s/%svideo-480p/media.m3u8", s.processedBucket, outputPrefix),
			Bitrate:    1000000,
		},
	}

	manifestURL := fmt.Sprintf("https://storage.googleapis.com/%s/%smanifest.m3u8", s.processedBucket, outputPrefix)

	now := time.Now().UTC()
	if err := s.videoRepo.UpdateProcessedVideos(ctx, videoID, processedVideos, manifestURL, &now); err != nil {
		log.Printf("[MONITORING] Failed to update processed videos for %s: %v", videoID, err)
		return
	}

	if err := s.videoRepo.UpdateStatus(ctx, videoID, models.StatusReady, nil); err != nil {
		log.Printf("[MONITORING] Failed to update status to ready for %s: %v", videoID, err)
		return
	}

	log.Printf("[MONITORING] Video %s processing complete and status updated to ready", videoID)
}

func (s *ProcessingService) handleJobFailure(ctx context.Context, videoID, errorMsg string) {
	log.Printf("[MONITORING] Job failed for video %s: %s", videoID, errorMsg)

	fullErrorMsg := fmt.Sprintf("Transcoding failed: %s", errorMsg)
	if err := s.videoRepo.UpdateStatus(ctx, videoID, models.StatusFailed, &fullErrorMsg); err != nil {
		log.Printf("[MONITORING] Failed to update status to failed for %s: %v", videoID, err)
	}
}
