package models

import "time"

type VideoUploadedEvent struct {
	VideoID    string    `json:"videoId"`
	ObjectName string    `json:"objectName"`
	Bucket     string    `json:"bucket"`
	FileSize   int64     `json:"fileSize"`
	MimeType   string    `json:"mimeType"`
	Timestamp  time.Time `json:"timestamp"`
}

type VideoProcessingCompleteEvent struct {
	VideoID         string            `json:"videoId"`
	JobID           string            `json:"jobId"`
	Status          string            `json:"status"`
	ProcessedVideos []ProcessedOutput `json:"processedVideos,omitempty"`
	ManifestURL     string            `json:"manifestUrl,omitempty"`
	DurationSeconds int               `json:"durationSeconds,omitempty"`
	ErrorMessage    string            `json:"errorMessage,omitempty"`
	Timestamp       time.Time         `json:"timestamp"`
}

type ProcessedOutput struct {
	Resolution string `json:"resolution"`
	ObjectName string `json:"objectName"`
	FileSize   int64  `json:"fileSize"`
	Bitrate    int    `json:"bitrate"`
}
