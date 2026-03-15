package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/controllers"
)

func RegisterUploadRoutes(r chi.Router, c *controller.UploadController) {
	r.Post("/videos/upload", c.HandleUpload)
	r.Post("/videos/upload/presign", c.HandlePresign)
	r.Post("/videos/{video_id}/upload/finalize", c.HandleFinalize)
}
