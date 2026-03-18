module github.com/LgAcerbi/go-video-upload/services/expirer

go 1.24

require (
	github.com/LgAcerbi/go-video-upload/pkg v0.0.0
	github.com/LgAcerbi/go-video-upload/proto/upload v0.0.0
	google.golang.org/grpc v1.68.0
)

require (
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/LgAcerbi/go-video-upload/pkg => ../../pkg

replace github.com/LgAcerbi/go-video-upload/proto/upload => ../../proto/upload
