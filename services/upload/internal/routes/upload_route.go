package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/controller"
)

func RegisterUploadRoutes(r chi.Router, c *controller.UploadController) {
	r.Post("/upload", c.HandleUpload)
}
