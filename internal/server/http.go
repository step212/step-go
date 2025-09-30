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
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/handlers"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	cd *conf.Data,
	cs *conf.Server,
	greeter *service.GreeterService,
	step *service.StepService,
	minio *service.MinioService,
	stepNoauth *service.StepNoauthService,
	portrait *service.PortraitService,
	feedback *service.FeedbackService,
	logger log.Logger,
) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
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

	opts = append(opts, http.Filter(
		handlers.CORS(
			handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
			handlers.AllowedOrigins([]string{"*"}),
		),
	))

	if cs.Http.Network != "" {
		opts = append(opts, http.Network(cs.Http.Network))
	}
	if cs.Http.Addr != "" {
		opts = append(opts, http.Address(cs.Http.Addr))
	}
	if cs.Http.Timeout != nil {
		opts = append(opts, http.Timeout(cs.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterGreeterHTTPServer(srv, greeter)
	stepApi.RegisterStepServiceHTTPServer(srv, step)
	minioApi.RegisterMinioHTTPServer(srv, minio)
	stepApi.RegisterStepNoauthServiceHTTPServer(srv, stepNoauth)
	stepApi.RegisterPortraitServiceHTTPServer(srv, portrait)
	stepApi.RegisterFeedbackServiceHTTPServer(srv, feedback)

	stepRoute := srv.Route("/step")
	stepRoute.POST("/upload", step.Upload)

	feedBackRoute := srv.Route("/feedback")
	feedBackRoute.POST("/award/create", feedback.CreateFeedbackAward)
	feedBackRoute.POST("/award/realize", feedback.RealizeFeedbackAward)

	return srv
}
