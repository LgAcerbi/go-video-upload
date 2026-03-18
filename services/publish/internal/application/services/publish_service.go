package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/LgAcerbi/go-video-upload/pkg/models"
	"github.com/LgAcerbi/go-video-upload/services/publish/internal/application/ports"
)

const stepPublish = "publish"

// bandwidthByResolution is a rough default bandwidth in bps for master playlist.
var bandwidthByResolution = map[string]int{
	"360p":  800000,
	"480p":  1400000,
	"720p":  2500000,
	"1080p": 5000000,
	"original": 4000000,
}

type PublishService struct {
	uploadClient ports.UploadStateClient
	uploader     ports.MasterPlaylistUploader
	stepPub      ports.StepResultPublisher
	bucket       string
}

func NewPublishService(
	uploadClient ports.UploadStateClient,
	uploader ports.MasterPlaylistUploader,
	stepPub ports.StepResultPublisher,
	bucket string,
) *PublishService {
	return &PublishService{
		uploadClient: uploadClient,
		uploader:     uploader,
		stepPub:      stepPub,
		bucket:       bucket,
	}
}

func (s *PublishService) Publish(ctx context.Context, uploadID string) error {
	ctxData, err := s.uploadClient.GetUploadProcessingContext(ctx, uploadID)
	if err != nil {
		s.reportFailed(ctx, uploadID, err)
		return err
	}
	videoID := ctxData.VideoID

	ready, err := s.uploadClient.ListReadyRenditions(ctx, videoID)
	if err != nil {
		s.reportFailed(ctx, uploadID, err)
		return err
	}
	if len(ready) == 0 {
		e := fmt.Errorf("no ready renditions to publish")
		s.reportFailed(ctx, uploadID, e)
		return e
	}

	// Build master.m3u8 content referencing variant playlists
	var b strings.Builder
	b.WriteString("#EXTM3U\n#EXT-X-VERSION:3\n")
	for _, rend := range ready {
		bw := bandwidthByResolution[rend.Resolution]
		if bw == 0 {
			bw = 2000000
		}
		// Resolution in format 720p -> 1280x720 we don't have here; use a placeholder or omit
		b.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d\n", bw))
		b.WriteString(rend.Resolution + "/playlist.m3u8\n")
	}
	masterContent := []byte(b.String())

	masterKey := "videos/" + videoID + "/hls/master.m3u8"
	if err := s.uploader.UploadMasterPlaylist(ctx, s.bucket, masterKey, masterContent); err != nil {
		s.reportFailed(ctx, uploadID, fmt.Errorf("upload master playlist: %w", err))
		return err
	}

	if err := s.uploadClient.UpdateVideoPlayback(ctx, videoID, masterKey); err != nil {
		s.reportFailed(ctx, uploadID, err)
		return err
	}
	if err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepPublish, "done", ""); err != nil {
		s.reportFailed(ctx, uploadID, err)
		return err
	}
	return s.stepPub.PublishStepResult(ctx, uploadID, stepPublish, "done", "")
}

func (s *PublishService) reportFailed(ctx context.Context, uploadID string, err error) {
	errMsg := err.Error()
	_ = s.uploadClient.UpdateUploadStep(ctx, uploadID, stepPublish, models.UploadStatusFailed, errMsg)
	_ = s.uploadClient.UpdateUploadStatus(ctx, uploadID, models.UploadStatusFailed)
	_ = s.stepPub.PublishStepResult(ctx, uploadID, stepPublish, models.UploadStatusFailed, errMsg)
}
