package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/zavieruka/video-platform/backend/internal/database"
	"github.com/zavieruka/video-platform/backend/internal/errors"
	"github.com/zavieruka/video-platform/backend/internal/models"
	"github.com/zavieruka/video-platform/backend/internal/storage"
	"github.com/zavieruka/video-platform/backend/internal/validation"
)

// VideoService defines the interface for video business logic
type VideoService interface {
	RequestUploadURL(ctx context.Context, req *models.UploadURLRequest) (*models.UploadURLResponse, error)
	ConfirmUpload(ctx context.Context, videoID string, req *models.ConfirmUploadRequest) (*models.Video, error)
	FailUpload(ctx context.Context, videoID string, req *models.FailUploadRequest) (*models.FailUploadResponse, error)
	GetVideo(ctx context.Context, videoID string) (*models.Video, error)
	ListVideos(ctx context.Context, limit, offset int) (*models.VideoListResponse, error)
}

// VideoServiceImpl implements VideoService
type VideoServiceImpl struct {
	repository      database.VideoRepository
	storage         storage.VideoStorage
	validator       *validation.VideoValidator
	uploadExpiryHrs int
}

// NewVideoService creates a new video service
func NewVideoService(
	repository database.VideoRepository,
	storage storage.VideoStorage,
	validator *validation.VideoValidator,
	uploadExpiryHrs int,
) *VideoServiceImpl {
	return &VideoServiceImpl{
		repository:      repository,
		storage:         storage,
		validator:       validator,
		uploadExpiryHrs: uploadExpiryHrs,
	}
}

// RequestUploadURL generates a signed upload URL and creates a pending video record
func (s *VideoServiceImpl) RequestUploadURL(ctx context.Context, req *models.UploadURLRequest) (*models.UploadURLResponse, error) {
	// Validate request
	if err := s.validator.ValidateUploadRequest(req); err != nil {
		return nil, err
	}

	// Generate unique video ID
	videoID := uuid.New().String()

	// Generate object name with UUID and original extension
	ext := filepath.Ext(req.FileName)
	objectName := fmt.Sprintf("videos/%s%s", videoID, ext)

	// Generate signed upload URL
	expiryDuration := time.Duration(s.uploadExpiryHrs) * time.Hour
	uploadURL, err := s.storage.GenerateSignedUploadURL(ctx, objectName, req.MimeType, expiryDuration)
	if err != nil {
		return nil, err
	}

	// Create pending video record
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
		StorageURL:         s.storage.(*storage.GCSVideoStorage).GetStorageURL(objectName),
		PublicURL:          s.storage.GetPublicURL(objectName),
		UploadURLExpiresAt: expiresAt,
		UploadedBy:         "", // Will be set when auth is implemented
		CreatedAt:          now,
		UpdatedAt:          now,
		LastError:          nil,
	}

	if err := s.repository.Create(ctx, video); err != nil {
		return nil, err
	}

	// Return response
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
		},
	}

	return response, nil
}

// ConfirmUpload verifies the file was uploaded and updates the video status
func (s *VideoServiceImpl) ConfirmUpload(ctx context.Context, videoID string, req *models.ConfirmUploadRequest) (*models.Video, error) {
	// Get the video record
	video, err := s.repository.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	// Check if video is in pending status
	if video.Status != models.StatusPending {
		return nil, errors.NewBadRequestError(fmt.Sprintf("Video is not in pending status (current status: %s)", video.Status))
	}

	// Check if upload URL has expired
	if time.Now().UTC().After(video.UploadURLExpiresAt) {
		errorMsg := "Upload URL has expired"
		if updateErr := s.repository.UpdateStatus(ctx, videoID, models.StatusFailed, &errorMsg); updateErr != nil {
			// Log but don't fail
			fmt.Printf("Failed to update status to failed: %v\n", updateErr)
		}
		return nil, errors.NewBadRequestError("Upload URL has expired. Please request a new upload URL.")
	}

	// Extract object name from storage URL
	// Format: gs://bucket/videos/uuid.ext
	objectName := video.StorageURL[len(fmt.Sprintf("gs://%s/", s.getBucketNameFromStorageURL(video.StorageURL))):]

	// Verify file exists in Cloud Storage
	exists, err := s.storage.FileExists(ctx, objectName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewBadRequestError("File not found in storage. Please ensure the file was uploaded successfully.")
	}

	// Verify file size matches
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

	// Update status to uploaded
	if err := s.repository.UpdateStatus(ctx, videoID, models.StatusUploaded, nil); err != nil {
		return nil, err
	}

	// Retrieve updated video
	video, err = s.repository.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	return video, nil
}

// FailUpload marks an upload as failed
func (s *VideoServiceImpl) FailUpload(ctx context.Context, videoID string, req *models.FailUploadRequest) (*models.FailUploadResponse, error) {
	// Get the video record
	video, err := s.repository.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}

	// Check if video is in pending status
	if video.Status != models.StatusPending {
		return nil, errors.NewBadRequestError(fmt.Sprintf("Video is not in pending status (current status: %s)", video.Status))
	}

	// Format error message
	errorMsg := fmt.Sprintf("%s: %s", req.Error, req.Message)

	// Update status to failed
	if err := s.repository.UpdateStatus(ctx, videoID, models.StatusFailed, &errorMsg); err != nil {
		return nil, err
	}

	// Return response
	response := &models.FailUploadResponse{
		ID:      videoID,
		Status:  models.StatusFailed,
		Message: "Upload marked as failed. You can retry by requesting a new upload URL.",
	}

	return response, nil
}

// GetVideo retrieves a video by ID
func (s *VideoServiceImpl) GetVideo(ctx context.Context, videoID string) (*models.Video, error) {
	return s.repository.GetByID(ctx, videoID)
}

// ListVideos retrieves a paginated list of videos
func (s *VideoServiceImpl) ListVideos(ctx context.Context, limit, offset int) (*models.VideoListResponse, error) {
	// Set default and max limits
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

	// Convert to response format
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

// Helper method to extract bucket name from storage URL
func (s *VideoServiceImpl) getBucketNameFromStorageURL(storageURL string) string {
	// Extract bucket name from gs://bucket/path format
	const prefix = "gs://"
	if len(storageURL) <= len(prefix) {
		return ""
	}
	path := storageURL[len(prefix):]
	for i, c := range path {
		if c == '/' {
			return path[:i]
		}
	}
	return path
}
