package service

import (
	"context"
	"fmt"

	v1 "step/api/minio/v1"
	"step/internal/biz"
)

type MinioService struct {
	v1.UnimplementedMinioServer

	uc *biz.MinioUsecase
}

func NewMinioService(uc *biz.MinioUsecase) *MinioService {
	return &MinioService{uc: uc}
}

func (s *MinioService) GetUploadPreSignedUrl(ctx context.Context, req *v1.GetUploadPreSignedUrlRequest) (*v1.GetUploadPreSignedUrlReply, error) {
	return nil, fmt.Errorf("hide back to step service")
	//return s.uc.GetUploadPreSignedUrl(ctx, req)
}

func (s *MinioService) GetDownloadPreSignedUrl(ctx context.Context, req *v1.GetDownloadPreSignedUrlRequest) (*v1.GetDownloadPreSignedUrlReply, error) {
	return nil, fmt.Errorf("hide back to step service")
	//return s.uc.GetDownloadPreSignedUrl(ctx, req)
}
