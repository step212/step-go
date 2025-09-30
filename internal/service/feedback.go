package service

import (
	"context"
	"step/internal/biz"

	stepApi "step/api/step/v1"

	"github.com/go-kratos/kratos/v2/transport/http"
)

type FeedbackService struct {
	stepApi.UnimplementedFeedbackServiceServer

	uc *biz.FeedbackUsecase
}

func NewFeedbackService(uc *biz.FeedbackUsecase) *FeedbackService {
	return &FeedbackService{uc: uc}
}

func (s *FeedbackService) CreateFeedbackAward(ctx http.Context) error {
	return s.uc.CreateFeedbackAward(ctx)
}

func (s *FeedbackService) RealizeFeedbackAward(ctx http.Context) error {
	return s.uc.RealizeFeedbackAward(ctx)
}

func (s *FeedbackService) GetFeedbackAwards(ctx context.Context, req *stepApi.GetFeedbackAwardsRequest) (*stepApi.GetFeedbackAwardsReply, error) {
	return s.uc.GetFeedbackAwards(ctx, req)
}

func (s *FeedbackService) GetFeedbackAward(ctx context.Context, req *stepApi.GetFeedbackAwardRequest) (*stepApi.GetFeedbackAwardReply, error) {
	return s.uc.GetFeedbackAward(ctx, req)
}

func (s *FeedbackService) DeleteFeedbackAward(ctx context.Context, req *stepApi.DeleteFeedbackAwardRequest) (*stepApi.DeleteFeedbackAwardReply, error) {
	return s.uc.DeleteFeedbackAward(ctx, req)
}
