package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/services"
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
		if errors.Is(err, domain.ErrInvalidExtension) {
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
