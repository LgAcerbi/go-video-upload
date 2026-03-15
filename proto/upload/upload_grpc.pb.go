package upload

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// UploadStateServiceServer is implemented by the upload service. Workers call it via gRPC to update state.
type UploadStateServiceServer interface {
	UpdateUploadStatus(context.Context, *UpdateUploadStatusRequest) (*UpdateUploadStatusResponse, error)
	UpdateUploadStep(context.Context, *UpdateUploadStepRequest) (*UpdateUploadStepResponse, error)
	UpdateVideoMetadata(context.Context, *UpdateVideoMetadataRequest) (*UpdateVideoMetadataResponse, error)
	CreateUploadSteps(context.Context, *CreateUploadStepsRequest) (*CreateUploadStepsResponse, error)
}

// UnimplementedUploadStateServiceServer must be embedded for forward compatibility.
type UnimplementedUploadStateServiceServer struct{}

func (UnimplementedUploadStateServiceServer) UpdateUploadStatus(context.Context, *UpdateUploadStatusRequest) (*UpdateUploadStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateUploadStatus not implemented")
}
func (UnimplementedUploadStateServiceServer) UpdateUploadStep(context.Context, *UpdateUploadStepRequest) (*UpdateUploadStepResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateUploadStep not implemented")
}
func (UnimplementedUploadStateServiceServer) UpdateVideoMetadata(context.Context, *UpdateVideoMetadataRequest) (*UpdateVideoMetadataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateVideoMetadata not implemented")
}
func (UnimplementedUploadStateServiceServer) CreateUploadSteps(context.Context, *CreateUploadStepsRequest) (*CreateUploadStepsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUploadSteps not implemented")
}

// UnsafeUploadStateServiceServer may be embedded to opt out of forward compatibility.
type UnsafeUploadStateServiceServer interface {
	mustEmbedUnimplementedUploadStateServiceServer()
}

func (UnimplementedUploadStateServiceServer) mustEmbedUnimplementedUploadStateServiceServer() {}

type uploadStateServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewUploadStateServiceClient(cc grpc.ClientConnInterface) UploadStateServiceClient {
	return &uploadStateServiceClient{cc}
}

type UploadStateServiceClient interface {
	UpdateUploadStatus(ctx context.Context, in *UpdateUploadStatusRequest, opts ...grpc.CallOption) (*UpdateUploadStatusResponse, error)
	UpdateUploadStep(ctx context.Context, in *UpdateUploadStepRequest, opts ...grpc.CallOption) (*UpdateUploadStepResponse, error)
	UpdateVideoMetadata(ctx context.Context, in *UpdateVideoMetadataRequest, opts ...grpc.CallOption) (*UpdateVideoMetadataResponse, error)
	CreateUploadSteps(ctx context.Context, in *CreateUploadStepsRequest, opts ...grpc.CallOption) (*CreateUploadStepsResponse, error)
}

func (c *uploadStateServiceClient) UpdateUploadStatus(ctx context.Context, in *UpdateUploadStatusRequest, opts ...grpc.CallOption) (*UpdateUploadStatusResponse, error) {
	out := new(UpdateUploadStatusResponse)
	err := c.cc.Invoke(ctx, "/upload.UploadStateService/UpdateUploadStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *uploadStateServiceClient) UpdateUploadStep(ctx context.Context, in *UpdateUploadStepRequest, opts ...grpc.CallOption) (*UpdateUploadStepResponse, error) {
	out := new(UpdateUploadStepResponse)
	err := c.cc.Invoke(ctx, "/upload.UploadStateService/UpdateUploadStep", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *uploadStateServiceClient) UpdateVideoMetadata(ctx context.Context, in *UpdateVideoMetadataRequest, opts ...grpc.CallOption) (*UpdateVideoMetadataResponse, error) {
	out := new(UpdateVideoMetadataResponse)
	err := c.cc.Invoke(ctx, "/upload.UploadStateService/UpdateVideoMetadata", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *uploadStateServiceClient) CreateUploadSteps(ctx context.Context, in *CreateUploadStepsRequest, opts ...grpc.CallOption) (*CreateUploadStepsResponse, error) {
	out := new(CreateUploadStepsResponse)
	err := c.cc.Invoke(ctx, "/upload.UploadStateService/CreateUploadSteps", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func RegisterUploadStateServiceServer(s grpc.ServiceRegistrar, srv UploadStateServiceServer) {
	s.RegisterService(&UploadStateService_ServiceDesc, srv)
}

func _UploadStateService_UpdateUploadStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateUploadStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UploadStateServiceServer).UpdateUploadStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/upload.UploadStateService/UpdateUploadStatus"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UploadStateServiceServer).UpdateUploadStatus(ctx, req.(*UpdateUploadStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UploadStateService_UpdateUploadStep_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateUploadStepRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UploadStateServiceServer).UpdateUploadStep(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/upload.UploadStateService/UpdateUploadStep"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UploadStateServiceServer).UpdateUploadStep(ctx, req.(*UpdateUploadStepRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UploadStateService_UpdateVideoMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateVideoMetadataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UploadStateServiceServer).UpdateVideoMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/upload.UploadStateService/UpdateVideoMetadata"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UploadStateServiceServer).UpdateVideoMetadata(ctx, req.(*UpdateVideoMetadataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UploadStateService_CreateUploadSteps_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateUploadStepsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UploadStateServiceServer).CreateUploadSteps(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/upload.UploadStateService/CreateUploadSteps"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UploadStateServiceServer).CreateUploadSteps(ctx, req.(*CreateUploadStepsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var UploadStateService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "upload.UploadStateService",
	HandlerType: (*UploadStateServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "UpdateUploadStatus", Handler: _UploadStateService_UpdateUploadStatus_Handler},
		{MethodName: "UpdateUploadStep", Handler: _UploadStateService_UpdateUploadStep_Handler},
		{MethodName: "UpdateVideoMetadata", Handler: _UploadStateService_UpdateVideoMetadata_Handler},
		{MethodName: "CreateUploadSteps", Handler: _UploadStateService_CreateUploadSteps_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "upload.proto",
}
