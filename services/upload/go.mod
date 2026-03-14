module github.com/LgAcerbi/go-video-upload/services/upload

go 1.24

require (
	github.com/aws/aws-sdk-go-v2 v1.32.6
	github.com/aws/aws-sdk-go-v2/config v1.28.6
	github.com/aws/aws-sdk-go-v2/credentials v1.17.44
	github.com/aws/aws-sdk-go-v2/service/s3 v1.80.0
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/LgAcerbi/go-video-upload/pkg v0.0.0
)

replace github.com/LgAcerbi/go-video-upload/pkg => ../../pkg
