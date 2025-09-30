package server

import (
	v1 "step/api/helloworld/v1"
	minioApi "step/api/minio/v1"
	stepApi "step/api/step/v1"
	"step/internal/conf"
	"step/internal/service"
	selfTrace "step/pkg/middleware/trace"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *conf.Server,
	greeter *service.GreeterService,
	step *service.StepService,
	minio *service.MinioService,
	stepNoauth *service.StepNoauthService,
	portrait *service.PortraitService,
	feedback *service.FeedbackService,
	logger log.Logger,
) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			logging.Server(logger),
			metadata.Server(),
			metrics.Server(),
			validate.Validator(),
			selfTrace.MetaServer(),
			selfTrace.Server(),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterGreeterServer(srv, greeter)
	stepApi.RegisterStepServiceServer(srv, step)
	minioApi.RegisterMinioServer(srv, minio)
	stepApi.RegisterStepNoauthServiceServer(srv, stepNoauth)
	stepApi.RegisterPortraitServiceServer(srv, portrait)
	stepApi.RegisterFeedbackServiceServer(srv, feedback)

	return srv
}
