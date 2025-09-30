package data

import (
	"context"
	"crypto/tls"
	"net/http"
	"step/internal/conf"
	"step/internal/data/ent"
	"time"

	"entgo.io/ent/dialect/sql"

	"step/pkg/middleware/trace"

	"ariga.io/sqlcomment"
	"entgo.io/ent/dialect"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-kratos/kratos/v2/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"github.com/hibiken/asynq"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewDriver,
	NewEntClient,
	NewData,
	NewGreeterRepo,
	NewMinioRepo,
	NewStepRepo,
	NewStepNoauthRepo,
	NewEncryptRepo,
	NewAsynqEnqueueRepo,
	NewStatisticsRepo,
)

// Data .
type Data struct {
	helper *log.Helper

	ent_client *ent.Client

	s3_client             *s3.Client
	presign_client        *s3.PresignClient
	minio_endpoint        string
	minio_endpoint_remote string
	minio_bucket_name     string

	secret string

	asynq_client *asynq.Client
}

// NewData .
func NewData(c *conf.Data, logger log.Logger, ent_client *ent.Client) (*Data, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "data/data"))

	/*** init for minio client ***/
	cp := credentials.StaticCredentialsProvider{
		Value: aws.Credentials{
			AccessKeyID:     c.Minio.AccessKeyId,
			SecretAccessKey: c.Minio.SecretAccessKey,
		},
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	cfg := aws.Config{
		Credentials:  cp,
		BaseEndpoint: aws.String(c.Minio.Endpoint),
		Region:       "auto",
		HTTPClient:   &http.Client{Transport: tr},
	}
	s3Client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(s3Client)

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       int(c.Redis.AsynqDb),
	})

	cleanup := func() {
		helper.Info("closing the data resources")
		asynqClient.Close()
	}
	return &Data{
		helper:                helper,
		ent_client:            ent_client,
		s3_client:             s3Client,
		presign_client:        presignClient,
		minio_endpoint:        c.Minio.Endpoint,
		minio_endpoint_remote: c.Minio.EndpointRemote,
		minio_bucket_name:     c.Minio.BucketName,
		secret:                c.Secret,
		asynq_client:          asynqClient,
	}, cleanup, nil
}

func NewEntClient(drv dialect.Driver, logger log.Logger) *ent.Client {
	logs := log.NewHelper(log.With(logger, "module", "repo/data"))
	client := ent.NewClient(ent.Driver(drv))
	// auto 自动迁移
	err := client.Schema.Create(context.Background())
	if err != nil {
		logs.Fatalf("failed opening connection to db: %v", err)
	}
	logs.Info("database automatic migration success!")
	return client
}

func NewDriver(cd *conf.Data) dialect.Driver {
	db, err := sql.Open(
		cd.Database.Driver,
		cd.Database.Source,
	)
	if err != nil {
		log.Fatalf("failed opening connection to db: %v", err)
		panic(err)
	}

	db.DB().SetMaxIdleConns(int(cd.Database.MaxIdleConns))
	db.DB().SetMaxOpenConns(int(cd.Database.MaxOpenConns))
	db.DB().SetConnMaxLifetime(time.Duration(cd.Database.MaxConnLifeSeconds) * time.Second)

	var logdb dialect.Driver = db
	/* if cfg.Env.Mode == "dev" {
		//logdb = dialect.Debug(db)
	} */
	return sqlcomment.NewDriver(logdb,
		sqlcomment.WithTagger(
			// set trace_id in comment
			trace.TraceIDCommenter{},
		),
		sqlcomment.WithTags(
			sqlcomment.Tags{
				sqlcomment.KeyApplication: "step-go",
			},
		),
	)
}
