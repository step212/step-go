package biz

import (
	"context"

	stepApi "step/api/step/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type EncryptRepo interface {
	Encrypt(ctx context.Context, stepId uint64, data string) (string, error)
	Decrypt(ctx context.Context, stepId uint64, data string) (string, error)
}

type StepNoauthRepo interface {
	GetNoauthStep(ctx context.Context, req *stepApi.GetNoauthStepRequest) (*stepApi.GetNoauthStepReply, error)
	SetCommentForStep(ctx context.Context, req *stepApi.SetCommentForStepRequest) (*stepApi.SetCommentForStepReply, error)
}

type StepNoauthUsecase struct {
	repo             StepNoauthRepo
	log              *log.Helper
	encryptRepo      EncryptRepo
	asynqEnqueueRepo AsynqEnqueueRepo
}

func NewStepNoauthUsecase(
	repo StepNoauthRepo,
	encryptRepo EncryptRepo,
	asynqEnqueueRepo AsynqEnqueueRepo,
	logger log.Logger,
) *StepNoauthUsecase {
	return &StepNoauthUsecase{
		repo:             repo,
		encryptRepo:      encryptRepo,
		asynqEnqueueRepo: asynqEnqueueRepo,
		log:              log.NewHelper(logger, log.WithMessageKey("stepNoauthUsecase")),
	}
}

func (uc *StepNoauthUsecase) GetNoauthStep(ctx context.Context, req *stepApi.GetNoauthStepRequest) (*stepApi.GetNoauthStepReply, error) {
	return uc.repo.GetNoauthStep(ctx, req)
}

func (uc *StepNoauthUsecase) SetCommentForStep(ctx context.Context, req *stepApi.SetCommentForStepRequest) (*stepApi.SetCommentForStepReply, error) {
	reply, err := uc.repo.SetCommentForStep(ctx, req)
	if err != nil {
		return nil, err
	}

	err = uc.asynqEnqueueRepo.EnqueueStepComment(ctx, reply.Id, req.Type)
	if err != nil {
		uc.log.Errorf("EnqueueStepComment error: %v", err)
		//return err
	}

	return reply, nil
}

func (uc *StepNoauthUsecase) Decrypt(ctx context.Context, req *stepApi.DecryptRequest) (*stepApi.DecryptReply, error) {
	decrypted, err := uc.encryptRepo.Decrypt(ctx, req.Id, req.Data)
	if err != nil {
		return nil, err
	}
	return &stepApi.DecryptReply{Data: decrypted}, nil
}
