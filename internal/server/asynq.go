package server

import (
	"context"
	"step/internal/biz"
	"step/internal/conf"
	"step/internal/objects"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

type AsynqServer struct {
	log         *log.Helper
	asynqServer *asynq.Server
	mux         *asynq.ServeMux
}

// NewAsynqServer new an asynq server.
func NewAsynqServer(
	dc *conf.Data,
	logger log.Logger,
	asynqStatisticsUsecase *biz.AsynqStatisticsUsecase,
	asynqFeedbackUsecase *biz.AsynqFeedbackUsecase,
) *AsynqServer {
	log := log.NewHelper(logger, log.WithMessageKey("asynqSrv msg"))

	asynqServer := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     dc.Redis.Addr,
			Password: dc.Redis.Password,
			DB:       int(dc.Redis.AsynqDb),
		},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optionally specify multiple queues with different priority.
			Queues: objects.Queues,
			// See the godoc for other configuration options
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(objects.TypeTargetCreate, asynqStatisticsUsecase.HandleTargetCreate)
	mux.HandleFunc(objects.TypeStepCreate, asynqStatisticsUsecase.HandleStepCreate)
	mux.HandleFunc(objects.TypeStepComment, asynqStatisticsUsecase.HandleStepComment)
	mux.HandleFunc(objects.TypeFeedbackPortraitChange, asynqFeedbackUsecase.HandleFeedbackPortraitChange)

	return &AsynqServer{
		log:         log,
		asynqServer: asynqServer,
		mux:         mux,
	}
}

func (qs *AsynqServer) Start(ctx context.Context) error {
	return qs.asynqServer.Run(qs.mux)
}

func (qs *AsynqServer) Stop(ctx context.Context) error {
	qs.asynqServer.Stop()
	qs.asynqServer.Shutdown()
	return nil
}
