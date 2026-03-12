package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/zavieruka/video-platform/backend/internal/errors"
	"github.com/zavieruka/video-platform/backend/internal/models"
	"github.com/zavieruka/video-platform/backend/internal/services"
)

type VideoHandler struct {
	videoService services.VideoService
}

func NewVideoHandler(videoService services.VideoService) *VideoHandler {
	return &VideoHandler{
		videoService: videoService,
	}
}

func (h *VideoHandler) RequestUploadURL(w http.ResponseWriter, r *http.Request) {
	var req models.UploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	response, err := h.videoService.RequestUploadURL(r.Context(), &req)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *VideoHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("id")
	if videoID == "" {
		h.respondError(w, errors.NewBadRequestError("Video ID is required"))
		return
	}

	var req models.ConfirmUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = models.ConfirmUploadRequest{}
	}

	video, err := h.videoService.ConfirmUpload(r.Context(), videoID, &req)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, video.ToResponse())
}

func (h *VideoHandler) FailUpload(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("id")
	if videoID == "" {
		h.respondError(w, errors.NewBadRequestError("Video ID is required"))
		return
	}

	var req models.FailUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	response, err := h.videoService.FailUpload(r.Context(), videoID, &req)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *VideoHandler) GetVideo(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("id")
	if videoID == "" {
		h.respondError(w, errors.NewBadRequestError("Video ID is required"))
		return
	}

	video, err := h.videoService.GetVideo(r.Context(), videoID)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, video.ToResponse())
}

func (h *VideoHandler) ListVideos(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	offset := 0 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsedOffset
		}
	}

	response, err := h.videoService.ListVideos(r.Context(), limit, offset)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *VideoHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *VideoHandler) respondError(w http.ResponseWriter, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		// If it's not an AppError, wrap it as internal error
		appErr = errors.NewInternalError("An unexpected error occurred", err)
	}

	h.respondJSON(w, appErr.StatusCode, appErr)
}

func (h *VideoHandler) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("id")
	if videoID == "" {
		h.respondError(w, errors.NewBadRequestError("Video ID is required"))
		return
	}

	if err := h.videoService.DeleteVideo(r.Context(), videoID); err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "Video deleted successfully"})
}
