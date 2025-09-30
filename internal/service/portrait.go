package service

import (
	"context"
	stepApi "step/api/step/v1"
	"step/internal/biz"
)

type PortraitService struct {
	stepApi.UnimplementedPortraitServiceServer

	uc *biz.PortraitUsecase
}

func NewPortraitService(uc *biz.PortraitUsecase) *PortraitService {
	return &PortraitService{uc: uc}
}

func (s *PortraitService) GetPortraitBasic(ctx context.Context, req *stepApi.GetPortraitBasicRequest) (*stepApi.GetPortraitBasicReply, error) {
	return s.uc.GetPortraitBasic(ctx, req)
}

func (s *PortraitService) GetPortraitStepRate(ctx context.Context, req *stepApi.GetPortraitStepRateRequest) (*stepApi.GetPortraitStepRateReply, error) {
	return s.uc.GetPortraitStepRate(ctx, req)
}


