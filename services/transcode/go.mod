module github.com/LgAcerbi/go-video-upload/services/transcode

go 1.24

require (
	github.com/LgAcerbi/go-video-upload/pkg v0.0.0
	github.com/LgAcerbi/go-video-upload/proto/upload v0.0.0
	google.golang.org/grpc v1.68.0
)

require (
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.28.6
	github.com/aws/aws-sdk-go-v2/credentials v1.17.47
	github.com/aws/aws-sdk-go-v2/service/s3 v1.80.0
)

replace (
	github.com/LgAcerbi/go-video-upload/pkg => ../../pkg
	github.com/LgAcerbi/go-video-upload/proto/upload => ../../proto/upload
)
