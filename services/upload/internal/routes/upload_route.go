package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/controllers"
)

func RegisterUploadRoutes(r chi.Router, c *controller.UploadController) {
	r.Post("/upload", c.HandleUpload)
	r.Post("/upload/presign", c.HandlePresign)
}
