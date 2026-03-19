module github.com/LgAcerbi/go-video-upload/services/outbox-dispatcher

go 1.24

require (
	github.com/LgAcerbi/go-video-upload/pkg v0.0.0
	github.com/jackc/pgx/v5 v5.7.2
)

replace github.com/LgAcerbi/go-video-upload/pkg => ../../pkg
