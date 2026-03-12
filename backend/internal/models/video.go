package models

import "time"

type VideoStatus string

const (
	StatusPending    VideoStatus = "pending"
	StatusUploaded   VideoStatus = "uploaded"
	StatusFailed     VideoStatus = "failed"
	StatusProcessing VideoStatus = "processing"
	StatusReady      VideoStatus = "ready"
)

type Video struct {
	ID                 string      `firestore:"id"`
	Title              string      `firestore:"title"`
	Description        string      `firestore:"description"`
	FileName           string      `firestore:"fileName"`
	FileSize           int64       `firestore:"fileSize"`
	MimeType           string      `firestore:"mimeType"`
	Status             VideoStatus `firestore:"status"`
	ObjectName         string      `firestore:"objectName"`
	StorageURL         string      `firestore:"storageUrl"`
	PublicURL          string      `firestore:"publicUrl"`
	UploadURLExpiresAt time.Time   `firestore:"uploadUrlExpiresAt"`
	UploadedBy         string      `firestore:"uploadedBy"`
	CreatedAt          time.Time   `firestore:"createdAt"`
	UpdatedAt          time.Time   `firestore:"updatedAt"`
	LastError          *string     `firestore:"lastError,omitempty"`

	ProcessingJobID     *string                   `firestore:"processingJobId,omitempty"`
	ProcessedVideos     map[string]ProcessedVideo `firestore:"processedVideos,omitempty"`
	ProcessingStartedAt *time.Time                `firestore:"processingStartedAt,omitempty"`
	ProcessingEndedAt   *time.Time                `firestore:"processingEndedAt,omitempty"`
	DurationSeconds     int                       `firestore:"durationSeconds,omitempty"`
	ManifestURL         *string                   `firestore:"manifestUrl,omitempty"`
}

type ProcessedVideo struct {
	Resolution string `firestore:"resolution" json:"resolution"`
	StorageURL string `firestore:"storageUrl" json:"storageUrl"`
	PublicURL  string `firestore:"publicUrl" json:"publicUrl"`
	FileSize   int64  `firestore:"fileSize" json:"fileSize"`
	Bitrate    int    `firestore:"bitrate" json:"bitrate"`
}

// Request Types

type UploadURLRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	FileName    string `json:"fileName"`
	FileSize    int64  `json:"fileSize"`
	MimeType    string `json:"mimeType"`
}

type ConfirmUploadRequest struct {
	UploadedAt *time.Time `json:"uploadedAt,omitempty"`
}

type FailUploadRequest struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Response Types

type UploadURLResponse struct {
	VideoID   string            `json:"videoId"`
	UploadURL string            `json:"uploadUrl"`
	ExpiresAt time.Time         `json:"expiresAt"`
	Metadata  UploadURLMetadata `json:"metadata"`
}

type UploadURLMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	FileName    string `json:"fileName"`
	FileSize    int64  `json:"fileSize"`
	MimeType    string `json:"mimeType"`
	ObjectName  string `json:"objectName"`
}

type VideoResponse struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	FileName    string      `json:"fileName"`
	FileSize    int64       `json:"fileSize"`
	MimeType    string      `json:"mimeType"`
	Status      VideoStatus `json:"status"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	URLs        VideoURLs   `json:"urls"`
	LastError   *string     `json:"lastError,omitempty"`

	ProcessingStatus *ProcessingStatus        `json:"processingStatus,omitempty"`
	ProcessedVideos  []ProcessedVideoResponse `json:"processedVideos,omitempty"`
	ManifestURL      *string                  `json:"manifestUrl,omitempty"`
	DurationSeconds  int                      `json:"durationSeconds,omitempty"`
}

type ProcessingStatus struct {
	JobID           string     `json:"jobId"`
	StartedAt       time.Time  `json:"startedAt"`
	EndedAt         *time.Time `json:"endedAt,omitempty"`
	DurationSeconds float64    `json:"durationSeconds,omitempty"`
}

type ProcessedVideoResponse struct {
	Resolution string `json:"resolution"`
	URL        string `json:"url"`
	FileSize   int64  `json:"fileSize"`
	Bitrate    int    `json:"bitrate"`
}

type VideoURLs struct {
	Storage string `json:"storage"`
	Public  string `json:"public"`
}

type VideoListResponse struct {
	Videos     []VideoResponse `json:"videos"`
	TotalCount int             `json:"totalCount"`
	Limit      int             `json:"limit"`
	Offset     int             `json:"offset"`
}

type FailUploadResponse struct {
	ID      string      `json:"id"`
	Status  VideoStatus `json:"status"`
	Message string      `json:"message"`
}

func (v *Video) ToResponse() *VideoResponse {
	response := &VideoResponse{
		ID:          v.ID,
		Title:       v.Title,
		Description: v.Description,
		FileName:    v.FileName,
		FileSize:    v.FileSize,
		MimeType:    v.MimeType,
		Status:      v.Status,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
		URLs: VideoURLs{
			Storage: v.StorageURL,
			Public:  v.PublicURL,
		},
		LastError:       v.LastError,
		DurationSeconds: v.DurationSeconds,
		ManifestURL:     v.ManifestURL,
	}

	if v.ProcessingJobID != nil && v.ProcessingStartedAt != nil {
		processingStatus := &ProcessingStatus{
			JobID:     *v.ProcessingJobID,
			StartedAt: *v.ProcessingStartedAt,
			EndedAt:   v.ProcessingEndedAt,
		}

		if v.ProcessingEndedAt != nil {
			duration := v.ProcessingEndedAt.Sub(*v.ProcessingStartedAt).Seconds()
			processingStatus.DurationSeconds = duration
		}

		response.ProcessingStatus = processingStatus
	}

	if len(v.ProcessedVideos) > 0 {
		processedVideos := make([]ProcessedVideoResponse, 0, len(v.ProcessedVideos))
		for _, pv := range v.ProcessedVideos {
			processedVideos = append(processedVideos, ProcessedVideoResponse{
				Resolution: pv.Resolution,
				URL:        pv.PublicURL,
				FileSize:   pv.FileSize,
				Bitrate:    pv.Bitrate,
			})
		}
		response.ProcessedVideos = processedVideos
	}

	return response
}
