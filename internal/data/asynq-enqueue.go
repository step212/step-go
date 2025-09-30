package data

import (
	"context"
	"encoding/json"
	"step/internal/biz"
	"step/internal/objects"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

type asynqEnqueueRepo struct {
	data *Data
	log  *log.Helper
}

// NewAsynqEnqueueRepo .
func NewAsynqEnqueueRepo(data *Data, logger log.Logger) biz.AsynqEnqueueRepo {
	return &asynqEnqueueRepo{
		data: data,
		log:  log.NewHelper(logger, log.WithMessageKey("asynqEnqueueRepo")),
	}
}

func (r *asynqEnqueueRepo) EnqueueTargetCreate(ctx context.Context, targetID uint64) error {
	payload := objects.TargetCreatePayload{
		TargetID: targetID,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(objects.TypeTargetCreate, jsonPayload)

	_, err = r.data.asynq_client.Enqueue(task, asynq.Queue(objects.QueueDefault))
	return err
}

func (r *asynqEnqueueRepo) EnqueueStepCreate(ctx context.Context, stepID uint64) error {
	payload := objects.StepCreatePayload{
		StepID: stepID,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(objects.TypeStepCreate, jsonPayload)

	_, err = r.data.asynq_client.Enqueue(task, asynq.Queue(objects.QueueDefault))
	return err
}

func (r *asynqEnqueueRepo) EnqueueStepComment(ctx context.Context, stepID uint64, commentType string) error {
	payload := objects.StepCommentPayload{
		StepID:      stepID,
		CommentType: commentType,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(objects.TypeStepComment, jsonPayload)

	_, err = r.data.asynq_client.Enqueue(task, asynq.Queue(objects.QueueDefault))
	return err
}

func (r *asynqEnqueueRepo) EnqueueFeedbackPortraitChange(
	ctx context.Context,
	userID string,
	portraitChangeTypes []*objects.PortraitchangeType,
) error {
	payload := objects.FeedbackPortraitChangePayload{
		UserID: userID,
		PortraitChangeTypes: portraitChangeTypes,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(objects.TypeFeedbackPortraitChange, jsonPayload)

	_, err = r.data.asynq_client.Enqueue(task, asynq.Queue(objects.QueueDefault))
	return err
}