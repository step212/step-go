package service

import (
	"context"
	stepApi "step/api/step/v1"
	"step/internal/biz"
)

type StepNoauthService struct {
	stepApi.UnimplementedStepNoauthServiceServer

	uc *biz.StepNoauthUsecase
}

func NewStepNoauthService(uc *biz.StepNoauthUsecase) *StepNoauthService {
	return &StepNoauthService{uc: uc}
}

func (s *StepNoauthService) GetNoauthStep(ctx context.Context, req *stepApi.GetNoauthStepRequest) (*stepApi.GetNoauthStepReply, error) {
	return s.uc.GetNoauthStep(ctx, req)
}

func (s *StepNoauthService) SetCommentForStep(ctx context.Context, req *stepApi.SetCommentForStepRequest) (*stepApi.SetCommentForStepReply, error) {
	return s.uc.SetCommentForStep(ctx, req)
}

func (s *StepNoauthService) Decrypt(ctx context.Context, req *stepApi.DecryptRequest) (*stepApi.DecryptReply, error) {
	return s.uc.Decrypt(ctx, req)
}
