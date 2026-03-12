package services

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/zavieruka/video-platform/backend/internal/database"
	"github.com/zavieruka/video-platform/backend/internal/errors"
	"github.com/zavieruka/video-platform/backend/internal/models"
	"github.com/zavieruka/video-platform/backend/internal/pubsub"
	"github.com/zavieruka/video-platform/backend/internal/storage"
	"github.com/zavieruka/video-platform/backend/internal/validation"
)

type VideoService interface {
	RequestUploadURL(ctx context.Context, req *models.UploadURLRequest) (*models.UploadURLResponse, error)
	ConfirmUpload(ctx context.Context, videoID string, req *models.ConfirmUploadRequest) (*models.Video, error)
	FailUpload(ctx context.Context, videoID string, req *models.FailUploadRequest) (*models.FailUploadResponse, error)
	GetVideo(ctx context.Context, videoID string) (*models.Video, error)
	ListVideos(ctx context.Context, limit, offset int) (*models.VideoListResponse, error)
	DeleteVideo(ctx context.Context, videoID string) error
}

type VideoServiceImpl struct {
	repository        database.VideoRepository
	storage           storage.VideoStorage
	validator         *validation.VideoValidator
	uploadExpiryHrs   int
	publisher         *pubsub.Publisher
	sourceBucket      string
	enableAutoProcess bool
}

func NewVideoService(
	repository database.VideoRepository,
	storage storage.VideoStorage,
	validator *validation.VideoValidator,
	uploadExpiryHrs int,
	publisher *pubsub.Publisher,
	sourceBucket string,
	enableAutoProcess bool,
) *VideoServiceImpl {
	return &VideoServiceImpl{
		repository:        repository,
		storage:           storage,
		validator:         validator,
		uploadExpiryHrs:   uploadExpiryHrs,
		publisher:         publisher,
		sourceBucket:      sourceBucket,
		enableAutoProcess: enableAutoProcess,
	}
}

func (s *VideoServiceImpl) RequestUploadURL(ctx context.Context, req *models.UploadURLRequest) (*models.UploadURLResponse, error) {
	if err := s.validator.ValidateUploadRequest(req); err != nil {
		return nil, err
	}

	videoID := uuid.New().String()

	ext := filepath.Ext(req.FileName)
	objectName := fmt.Sprintf("videos/%s%s", videoID, ext)

	expiryDuration := time.Duration(s.uploadExpiryHrs) * time.Hour
	uploadURL, err := s.storage.GenerateSignedUploadURL(ctx, objectName, req.MimeType, expiryDuration)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(expiryDuration)

	video := &models.Video{
		ID:                 videoID,
		Title:              req.Title,
		Description:        req.Description,
		FileName:           req.FileName,
		FileSize:           req.FileSize,
		MimeType:           req.MimeType,
		Status:             models.StatusPending,
		ObjectName:         objectName,
		StorageURL:         s.storage.GetStorageURL(objectName),
		PublicURL:          s.storage.GetPublicURL(objectName),
		UploadURLExpiresAt: expiresAt,
		UploadedBy:         "",
		CreatedAt:          now,
		UpdatedAt:          now,
		LastError:          nil,
	}

	if err := s.repository.Create(ctx, video); err != nil {
		return nil, err
	}

	response := &models.UploadURLResponse{
		VideoID:   videoID,
		UploadURL: uploadURL,
		ExpiresAt: expiresAt,
		Metadata: models.UploadURLMetadata{
			Title:       req.Title,
			Description: req.Description,
			FileName:    req.FileName,
			FileSize:    req.FileSize,
			MimeType:    req.MimeType,
			ObjectName:  objectName,
		},
	}

	return response, nil
}

func (s *VideoServiceImpl) ConfirmUpload(ctx context.Context, videoID string, req *models.ConfirmUploadRequest) (*models.Video, error) {
	video, err := s.repository.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	if video.Status != models.StatusPending {
		return nil, errors.NewBadRequestError(fmt.Sprintf("Video is not in pending status (current status: %s)", video.Status))
	}

	if time.Now().UTC().After(video.UploadURLExpiresAt) {
		errorMsg := "Upload URL has expired"
		if updateErr := s.repository.UpdateStatus(ctx, videoID, models.StatusFailed, &errorMsg); updateErr != nil {
			fmt.Printf("Failed to update status to failed: %v\n", updateErr)
		}
		return nil, errors.NewBadRequestError("Upload URL has expired. Please request a new upload URL.")
	}

	objectName := video.ObjectName

	exists, err := s.storage.FileExists(ctx, objectName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewBadRequestError("File not found in storage. Please ensure the file was uploaded successfully.")
	}

	actualSize, err := s.storage.GetFileSize(ctx, objectName)
	if err != nil {
		return nil, err
	}
	if actualSize != video.FileSize {
		errorMsg := fmt.Sprintf("File size mismatch: expected %d bytes, got %d bytes", video.FileSize, actualSize)
		if updateErr := s.repository.UpdateStatus(ctx, videoID, models.StatusFailed, &errorMsg); updateErr != nil {
			fmt.Printf("Failed to update status to failed: %v\n", updateErr)
		}
		return nil, errors.NewBadRequestError(errorMsg)
	}

	if err := s.repository.UpdateStatus(ctx, videoID, models.StatusUploaded, nil); err != nil {
		return nil, err
	}

	if s.enableAutoProcess && s.publisher != nil {
		event := &models.VideoUploadedEvent{
			VideoID:    videoID,
			ObjectName: video.ObjectName,
			Bucket:     s.sourceBucket,
			FileSize:   video.FileSize,
			MimeType:   video.MimeType,
			Timestamp:  time.Now().UTC(),
		}

		if err := s.publisher.PublishVideoUploaded(ctx, event); err != nil {
			log.Printf("Failed to publish video uploaded event for video %s: %v", videoID, err)
			return nil, fmt.Errorf("failed to publish processing event: %w", err)
		}

		log.Printf("Published video uploaded event for video %s", videoID)
	}

	video, err = s.repository.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	return video, nil
}

func (s *VideoServiceImpl) FailUpload(ctx context.Context, videoID string, req *models.FailUploadRequest) (*models.FailUploadResponse, error) {
	video, err := s.repository.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	if video.Status != models.StatusPending {
		return nil, errors.NewBadRequestError(fmt.Sprintf("Video is not in pending status (current status: %s)", video.Status))
	}

	errorMsg := fmt.Sprintf("%s: %s", req.Error, req.Message)

	if err := s.repository.UpdateStatus(ctx, videoID, models.StatusFailed, &errorMsg); err != nil {
		return nil, err
	}

	response := &models.FailUploadResponse{
		ID:      videoID,
		Status:  models.StatusFailed,
		Message: "Upload marked as failed. You can retry by requesting a new upload URL.",
	}

	return response, nil
}

func (s *VideoServiceImpl) GetVideo(ctx context.Context, videoID string) (*models.Video, error) {
	return s.repository.GetByID(ctx, videoID)
}

func (s *VideoServiceImpl) ListVideos(ctx context.Context, limit, offset int) (*models.VideoListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	videos, totalCount, err := s.repository.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	videoResponses := make([]models.VideoResponse, len(videos))
	for i, video := range videos {
		videoResponses[i] = *video.ToResponse()
	}

	response := &models.VideoListResponse{
		Videos:     videoResponses,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}

	return response, nil
}

func (s *VideoServiceImpl) DeleteVideo(ctx context.Context, videoID string) error {
	video, err := s.repository.GetByID(ctx, videoID)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	switch video.Status {

	case models.StatusProcessing:
		return fmt.Errorf("Cannot delete video while processing")

	case models.StatusPending:
		// No object exists yet - safe

	case models.StatusUploaded:
		if err := s.storage.DeleteFile(ctx, video.ObjectName); err != nil {
			return fmt.Errorf("failed to delete storage object: %w", err)
		}

	case models.StatusReady:
		// TODO: delete processed artifacts

	case models.StatusFailed:
		_ = s.storage.DeleteFile(ctx, video.ObjectName)

	default:
		return fmt.Errorf("unsupported status: %s", video.Status)
	}

	return s.repository.Delete(ctx, videoID)
}
