package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/service"
)

type UploadController struct {
	svc *service.UploadService
}

func NewUploadController(svc *service.UploadService) *UploadController {
	return &UploadController{svc: svc}
}

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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"key": key})
}
