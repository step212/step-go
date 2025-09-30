package service

import (
	"context"
	stepApi "step/api/step/v1"
	"step/internal/biz"

	"github.com/go-kratos/kratos/v2/transport/http"
)

type StepService struct {
	stepApi.UnimplementedStepServiceServer

	uc *biz.StepUsecase
}

func NewStepService(uc *biz.StepUsecase) *StepService {
	return &StepService{uc: uc}
}

func (s *StepService) Upload(ctx http.Context) error {
	return s.uc.Upload(ctx)
}

func (s *StepService) CreateTarget(ctx context.Context, req *stepApi.CreateTargetRequest) (*stepApi.CreateTargetReply, error) {
	return s.uc.CreateTarget(ctx, req)
}

func (s *StepService) UpdateTarget(ctx context.Context, req *stepApi.UpdateTargetRequest) (*stepApi.UpdateTargetReply, error) {
	return s.uc.UpdateTarget(ctx, req)
}

func (s *StepService) GetTargets(ctx context.Context, req *stepApi.GetTargetsRequest) (*stepApi.GetTargetsReply, error) {
	return s.uc.GetTargets(ctx, req)
}

func (s *StepService) DeleteTarget(ctx context.Context, req *stepApi.DeleteTargetRequest) (*stepApi.DeleteTargetReply, error) {
	return s.uc.DeleteTarget(ctx, req)
}

func (s *StepService) DoneTarget(ctx context.Context, req *stepApi.DoneTargetRequest) (*stepApi.DoneTargetReply, error) {
	return s.uc.DoneTarget(ctx, req)
}

func (s *StepService) GetTarget(ctx context.Context, req *stepApi.GetTargetRequest) (*stepApi.GetTargetReply, error) {
	return s.uc.GetTarget(ctx, req)
}

func (s *StepService) GetTargetTree(ctx context.Context, req *stepApi.GetTargetTreeRequest) (*stepApi.GetTargetTreeReply, error) {
	return s.uc.GetTargetTree(ctx, req)
}

func (s *StepService) AddTargetDirStep(ctx context.Context, req *stepApi.AddTargetDirStepRequest) (*stepApi.AddTargetDirStepReply, error) {
	return s.uc.AddTargetDirStep(ctx, req)
}

func (s *StepService) GetTargetDirStepChildren(ctx context.Context, req *stepApi.GetTargetDirStepChildrenRequest) (*stepApi.GetTargetDirStepChildrenReply, error) {
	return s.uc.GetTargetDirStepChildren(ctx, req)
}

func (s *StepService) UpdateTargetStep(ctx context.Context, req *stepApi.UpdateTargetStepRequest) (*stepApi.UpdateTargetStepReply, error) {
	return s.uc.UpdateTargetStep(ctx, req)
}

func (s *StepService) DeleteTargetStep(ctx context.Context, req *stepApi.DeleteTargetStepRequest) (*stepApi.DeleteTargetStepReply, error) {
	return s.uc.DeleteTargetStep(ctx, req)
}

func (s *StepService) Encrypt(ctx context.Context, req *stepApi.EncryptRequest) (*stepApi.EncryptReply, error) {
	return s.uc.Encrypt(ctx, req)
}
