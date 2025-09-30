package biz

import (
	"context"
	"encoding/json"
	"step/internal/objects"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

type StatisticsRepo interface {
	HandleTargetCreate(ctx context.Context, task *asynq.Task) (uid string, portraitChangeTypes []*objects.PortraitchangeType, err error)
	HandleStepCreate(ctx context.Context, task *asynq.Task) (uid string, portraitChangeTypes []*objects.PortraitchangeType, err error)
	HandleStepComment(ctx context.Context, task *asynq.Task) (uid string, portraitChangeTypes []*objects.PortraitchangeType, err error)
}

// AsynqStatisticsUsecase is a AsynqStatistics usecase.
type AsynqStatisticsUsecase struct {
	log            *log.Helper
	statisticsRepo StatisticsRepo
	asynqEnqueueRepo AsynqEnqueueRepo
}

// NewAsynqStatisticsUsecase new a AsynqStatistics usecase.
func NewAsynqStatisticsUsecase(
	logger log.Logger,
	statisticsRepo StatisticsRepo,
	asynqEnqueueRepo AsynqEnqueueRepo,
) *AsynqStatisticsUsecase {
	return &AsynqStatisticsUsecase{
		log:            log.NewHelper(logger, log.WithMessageKey("asynqStatisticsUsecase")),
		statisticsRepo: statisticsRepo,
		asynqEnqueueRepo: asynqEnqueueRepo,
	}
}

func (uc *AsynqStatisticsUsecase) HandleTargetCreate(ctx context.Context, task *asynq.Task) error {
	var payload objects.TargetCreatePayload
	err := json.Unmarshal(task.Payload(), &payload)
	if err != nil {
		return err
	}

	uc.log.Infof("HandleTargetCreate: %v", payload)

	uid, portraitChangeTypes, err := uc.statisticsRepo.HandleTargetCreate(ctx, task)
	if err != nil {
		uc.log.Errorf("HandleTargetCreate: %v", err)
		return err
	}

	err = uc.asynqEnqueueRepo.EnqueueFeedbackPortraitChange(
		ctx, uid,
		portraitChangeTypes,
	)
	if err != nil {
		uc.log.Errorf("EnqueueFeedbackPortraitChange: %v", err)
		return err
	}

	return nil
}

func (uc *AsynqStatisticsUsecase) HandleStepCreate(ctx context.Context, task *asynq.Task) error {
	var payload objects.StepCreatePayload
	err := json.Unmarshal(task.Payload(), &payload)
	if err != nil {
		return err
	}

	uc.log.Infof("HandleStepCreate: %v", payload)

	uid, portraitChangeTypes, err := uc.statisticsRepo.HandleStepCreate(ctx, task)
	if err != nil {
		uc.log.Errorf("HandleStepCreate: %v", err)
		return err
	}

	err = uc.asynqEnqueueRepo.EnqueueFeedbackPortraitChange(
		ctx, uid,
		portraitChangeTypes,
	)
	if err != nil {
		uc.log.Errorf("EnqueueFeedbackPortraitChange: %v", err)
		return err
	}

	return nil
}

func (uc *AsynqStatisticsUsecase) HandleStepComment(ctx context.Context, task *asynq.Task) error {
	var payload objects.StepCommentPayload
	err := json.Unmarshal(task.Payload(), &payload)
	if err != nil {
		return err
	}

	uc.log.Infof("HandleStepComment: %v", payload)

	uid, portraitChangeTypes, err := uc.statisticsRepo.HandleStepComment(ctx, task)
	if err != nil {
		uc.log.Errorf("HandleStepComment: %v", err)
		return err
	}

	err = uc.asynqEnqueueRepo.EnqueueFeedbackPortraitChange(
		ctx, uid,
		portraitChangeTypes,
	)
	if err != nil {
		uc.log.Errorf("EnqueueFeedbackPortraitChange: %v", err)
		return err
	}

	return nil
}
