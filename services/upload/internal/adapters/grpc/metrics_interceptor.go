package grpcserver

import (
	"context"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxRequestFieldLen = 4096

func MetricsUnaryInterceptor(w *metrics.Writer, serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if w == nil {
			return handler(ctx, req)
		}
		start := time.Now()
		resp, err := handler(ctx, req)
		durationMs := time.Since(start).Milliseconds()
		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			} else {
				code = codes.Unknown
			}
		}
		fields := map[string]interface{}{
			"count":       1,
			"duration_ms": durationMs,
		}
		if msg, ok := req.(proto.Message); ok {
			if jsonBytes, marshalErr := protojson.Marshal(msg); marshalErr == nil {
				s := string(jsonBytes)
				if len(s) > maxRequestFieldLen {
					s = s[:maxRequestFieldLen] + "...(truncated)"
				}
				fields["request"] = s
			}
		}
		w.Record("grpc_request",
			map[string]string{
				"service": serviceName,
				"method":  info.FullMethod,
				"code":    code.String(),
			},
			fields,
		)
		return resp, err
	}
}
