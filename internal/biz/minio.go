package biz

import (
	"context"
	"mime/multipart"

	v1 "step/api/minio/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type MinioRepo interface {
	GetRemoteEndpoint(ctx context.Context) string
	GetUploadPreSignedUrl(ctx context.Context, upload_key string) (string, error)
	GetDownloadPreSignedUrl(ctx context.Context, download_key string, endpoint string) (string, error)
	UploadFile(ctx context.Context, upload_key string, file *multipart.File) error
	RemoveFile(ctx context.Context, remove_key string) error
}

type MinioUsecase struct {
	repo MinioRepo
	log  *log.Helper
}

func NewMinioUsecase(repo MinioRepo, logger log.Logger) *MinioUsecase {
	return &MinioUsecase{repo: repo, log: log.NewHelper(logger, log.WithMessageKey("minioUsecase"))}
}

func (uc *MinioUsecase) GetUploadPreSignedUrl(ctx context.Context, req *v1.GetUploadPreSignedUrlRequest) (*v1.GetUploadPreSignedUrlReply, error) {
	url, err := uc.repo.GetUploadPreSignedUrl(ctx, req.UploadKey)
	if err != nil {
		return nil, err
	}

	return &v1.GetUploadPreSignedUrlReply{
		Url: url,
	}, nil
}

func (uc *MinioUsecase) GetDownloadPreSignedUrl(ctx context.Context, req *v1.GetDownloadPreSignedUrlRequest) (*v1.GetDownloadPreSignedUrlReply, error) {
	url, err := uc.repo.GetDownloadPreSignedUrl(ctx, req.DownloadKey, "")
	if err != nil {
		return nil, err
	}

	return &v1.GetDownloadPreSignedUrlReply{
		Url: url,
	}, nil
}
