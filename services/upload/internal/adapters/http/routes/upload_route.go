package routes

import (
	controller "github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http"
	"github.com/go-chi/chi/v5"
)

func RegisterUploadRoutes(r chi.Router, c *controller.UploadController) {
	r.Post("/videos/upload", c.HandleUpload)
	r.Post("/videos/upload/presign", c.HandlePresign)
	r.Put("/videos/upload/put/{video_id}", c.HandlePutUploadProxy)
	r.Post("/videos/{video_id}/upload/finalize", c.HandleFinalize)
}
