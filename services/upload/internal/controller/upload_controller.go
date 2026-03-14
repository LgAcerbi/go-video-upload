package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/service"
)

type UploadResponse struct {
	Key string `json:"key"`
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
// @Router       /upload [post]
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
