package data

import (
	"context"
	"fmt"
	"mime/multipart"
	"step/internal/biz"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-kratos/kratos/v2/log"
)

type minioRepo struct {
	data *Data
	log  *log.Helper
}

func NewMinioRepo(data *Data, logger log.Logger) biz.MinioRepo {
	return &minioRepo{
		data: data,
		log:  log.NewHelper(logger, log.WithMessageKey("minioRepo")),
	}
}

func (m *minioRepo) GetRemoteEndpoint(ctx context.Context) string {
	return m.data.minio_endpoint_remote
}

func (m *minioRepo) GetUploadPreSignedUrl(ctx context.Context, upload_key string) (string, error) {
	presigned_request, err := m.data.presign_client.PresignPutObject(
		context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String(m.data.minio_bucket_name),
			Key:    aws.String(upload_key),
		},
		func(po *s3.PresignOptions) {
			po.Expires = time.Duration(time.Second * 1000)
		},
		s3.WithPresignClientFromClientOptions(
			func(o *s3.Options) {
				o.UsePathStyle = true
			},
		),
	)
	if err != nil {
		return "", fmt.Errorf("无法获取预签名url. 因为: %v\n", err)
	}

	return presigned_request.URL, nil
}

func (m *minioRepo) GetDownloadPreSignedUrl(ctx context.Context, download_key string, endpoint string) (string, error) {
	options := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = true
			if endpoint != "" {
				o.BaseEndpoint = aws.String(endpoint)
			}
		},
	}

	presigned_request, err := m.data.presign_client.PresignGetObject(
		context.TODO(),
		&s3.GetObjectInput{
			Bucket: aws.String(m.data.minio_bucket_name),
			Key:    aws.String(download_key),
		},
		func(po *s3.PresignOptions) {
			po.Expires = time.Duration(30 * time.Minute)
		},
		s3.WithPresignClientFromClientOptions(options...),
	)
	if err != nil {
		return "", fmt.Errorf("无法获取预签名url. 因为: %v\n", err)
	}

	return presigned_request.URL, nil
}

func (m *minioRepo) UploadFile(ctx context.Context, upload_key string, file *multipart.File) error {
	_, err := m.data.s3_client.PutObject(ctx,
		&s3.PutObjectInput{
			Bucket: aws.String(m.data.minio_bucket_name),
			Key:    aws.String(upload_key),
			Body:   *file,
		},
		func(po *s3.Options) {
			po.UsePathStyle = true
		},
	)
	if err != nil {
		return fmt.Errorf("无法上传文件%s. 因为: %v\n", upload_key, err)
	}

	return nil
}

func (m *minioRepo) RemoveFile(ctx context.Context, remove_key string) error {
	_, err := m.data.s3_client.DeleteObject(ctx,
		&s3.DeleteObjectInput{
			Bucket: aws.String(m.data.minio_bucket_name),
			Key:    aws.String(remove_key),
		},
		func(po *s3.Options) {
			po.UsePathStyle = true
		},
	)
	if err != nil {
		return fmt.Errorf("无法删除文件%s. 因为: %v\n", remove_key, err)
	}

	return nil
}
