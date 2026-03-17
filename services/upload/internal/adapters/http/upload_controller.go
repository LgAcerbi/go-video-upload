package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	service "github.com/LgAcerbi/go-video-upload/services/upload/internal/application/services"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain/entities"
	"github.com/go-chi/chi/v5"
)

type UploadResponse struct {
	Key string `json:"key"`
}

type PresignRequest struct {
	UserID string `json:"user_id"`
	Title  string `json:"title"`
}

type PresignResponse struct {
	UploadURL string `json:"upload_url"`
	VideoID   string `json:"video_id"`
}

type UploadController struct {
	svc    *service.UploadService
	logger logger.Logger
}

func NewUploadController(svc *service.UploadService, log logger.Logger) *UploadController {
	return &UploadController{svc: svc, logger: log}
}

// HandleUpload uploads a file via multipart form.
//
// @Summary      Upload a file
// @Description  Upload a file. Use multipart form with field name `file`. Accepted extensions are service-defined (e.g. video formats).
// @Tags         upload
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "File to upload"
// @Success      201  {object}  controller.UploadResponse  "Created, returns object key"
// @Failure      400  {string}  string  "Bad request (e.g. invalid extension)"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /videos/upload [post]
func (c *UploadController) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing or invalid file field (use form key 'file')", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	key, err := c.svc.UploadFile(r.Context(), header.Filename, file, header.Size, contentType)
	if err != nil {
		if errors.Is(err, service.ErrInvalidExtension) {
			c.logger.Info("upload rejected: invalid extension", "filename", header.Filename)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		c.logger.Error("upload failed", "filename", header.Filename, "error", err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	c.logger.Info("file uploaded", "key", key, "filename", header.Filename)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"key": key})
}

// HandlePresign returns a presigned URL for client-side upload. Creates video and upload records.
//
// @Summary      Request presigned upload URL
// @Description  Returns a presigned PUT URL and video_id. Client uploads the file with PUT to the URL. Body: user_id, title.
// @Tags         upload
// @Accept       json
// @Produce      json
// @Param        body  body  controller.PresignRequest  true  "user_id and title"
// @Success      200   {object}  controller.PresignResponse
// @Failure      400   {string}  string  "Bad request (e.g. missing user_id)"
// @Failure      500   {string}  string  "Internal server error"
// @Router       /videos/upload/presign [post]
func (c *UploadController) HandlePresign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req PresignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	uploadURL, videoID, err := c.svc.RequestPresignURL(r.Context(), req.UserID, req.Title)
	if err != nil {
		c.logger.Error("presign failed", "user_id", req.UserID, "error", err)
		if errors.Is(err, service.ErrInvalidPresignRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to create presigned URL", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(PresignResponse{UploadURL: uploadURL, VideoID: videoID})
}

// HandleFinalize confirms the client finished uploading to the presigned URL and enqueues the upload for processing.
//
// @Summary      Finalize upload
// @Description  Call after the client has uploaded the file to the presigned URL. Updates upload (storage_path, status) and sends event to upload-process queue.
// @Tags         upload
// @Produce      json
// @Param        video_id  path  string  true  "Video ID"
// @Success      200  "OK"
// @Failure      400  {string}  string  "Bad request (e.g. upload not found or not pending)"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /videos/{video_id}/upload/finalize [post]
func (c *UploadController) HandleFinalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	videoID := chi.URLParam(r, "video_id")
	if videoID == "" {
		http.Error(w, "video_id is required", http.StatusBadRequest)
		return
	}
	if err := c.svc.FinalizeUpload(r.Context(), videoID); err != nil {
		c.logger.Error("finalize failed", "video_id", videoID, "error", err)
		if errors.Is(err, service.ErrFinalizeUpload) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to finalize upload", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func parseLimit(r *http.Request, defaultLimit int) int {
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return defaultLimit
}

// ListUploadsResponseItem is the JSON shape for one upload in list responses.
type ListUploadsResponseItem struct {
	ID          string  `json:"id"`
	VideoID     string  `json:"video_id"`
	StoragePath string  `json:"storage_path"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

// ListVideosResponseItem is the JSON shape for one video in list responses.
type ListVideosResponseItem struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Title       string   `json:"title"`
	Format      string   `json:"format"`
	Status      string   `json:"status"`
	DurationSec *float64 `json:"duration_sec,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// HandleListUploads returns all uploads (GET /uploads).
func (c *UploadController) HandleListUploads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := parseLimit(r, 100)
	list, err := c.svc.ListUploads(r.Context(), limit)
	if err != nil {
		c.logger.Error("list uploads failed", "error", err)
		http.Error(w, "failed to list uploads", http.StatusInternalServerError)
		return
	}
	items := make([]ListUploadsResponseItem, len(list))
	for i, u := range list {
		items[i] = uploadToResponseItem(u)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"uploads": items})
}

// HandleListVideos returns all videos (GET /videos).
func (c *UploadController) HandleListVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := parseLimit(r, 100)
	list, err := c.svc.ListVideos(r.Context(), limit)
	if err != nil {
		c.logger.Error("list videos failed", "error", err)
		http.Error(w, "failed to list videos", http.StatusInternalServerError)
		return
	}
	items := make([]ListVideosResponseItem, len(list))
	for i, v := range list {
		items[i] = videoToResponseItem(v)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"videos": items})
}

func uploadToResponseItem(u *entities.Upload) ListUploadsResponseItem {
	item := ListUploadsResponseItem{
		ID:          u.ID,
		VideoID:     u.VideoID,
		StoragePath: u.StoragePath,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if u.ExpiresAt != nil {
		s := u.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		item.ExpiresAt = &s
	}
	return item
}

func videoToResponseItem(v *entities.Video) ListVideosResponseItem {
	return ListVideosResponseItem{
		ID:          v.ID,
		UserID:      v.UserID,
		Title:       v.Title,
		Format:      v.Format,
		Status:      v.Status,
		DurationSec: v.DurationSec,
		CreatedAt:   v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   v.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
