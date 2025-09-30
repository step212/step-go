package biz

import (
	"context"
	"errors"
	"fmt"
	"step/internal/conf"
	"step/internal/data/ent"
	"step/internal/data/ent/award"
	"step/internal/utils"
	"time"

	stepApi "step/api/step/v1"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-kratos/kratos/v2/log"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/spf13/cast"
)

type FeedbackUsecase struct {
	stepRepo         StepRepo
	minioRepo        MinioRepo
	entClient        *ent.Client
	minio_endpoint_remote   string
	log              *log.Helper
}

func NewFeedbackUsecase(
	stepRepo StepRepo,
	minioRepo MinioRepo,
	entClient *ent.Client,
	dataConf *conf.Data,
	logger log.Logger,
) *FeedbackUsecase {
	return &FeedbackUsecase{
		stepRepo:         stepRepo,
		minioRepo:        minioRepo,
		entClient:        entClient,
		minio_endpoint_remote:   dataConf.Minio.EndpointRemote,
		log:              log.NewHelper(logger, log.WithMessageKey("feedbackUsecase")),
	}
}

func (uc *FeedbackUsecase) CreateFeedbackAward(ctx kratosHttp.Context) error {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return errors.New("uid is empty")
	}

	req := ctx.Request()
	
	// 解析 multipart form 以处理多个文件
	err := req.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		return errors.New("failed to parse multipart form: " + err.Error())
	}

	// 获取所有上传的文件
	multipartForm := req.MultipartForm
	files, _ := multipartForm.File["files"]
	// 允许没有文件
	/* if !exists || len(files) == 0 {
		return errors.New("no files provided")
	} */

	description := req.FormValue("description")
	targetType := req.FormValue("targetType")
	scope := req.FormValue("scope")
	dimension := req.FormValue("dimension")
	threshold := req.FormValue("threshold")
	now := time.Now().Local()

	uc.log.Infof("create feedback award with %d files: description=%s, targetType=%s, scope=%s, dimension=%s, threshold=%s, time=%s", 
		len(files), description, targetType, scope, dimension, threshold, now)

	feedbackAward, err := uc.entClient.Award.Create().
		SetUserID(uid).
		SetStatus(award.StatusSetted).
		SetDescription(description).
		SetTargetType(award.TargetType(targetType)).
		SetScope(scope).
		SetDimension(dimension).
		SetThreshold(cast.ToInt32(threshold)).
		SetSettedAt(now.Unix()).
		Save(ctx)
	if err != nil {
		return err
	}

	// 处理每个文件
	var processedFiles []string
	for i, fileHeader := range files {
		// 打开文件
		file, err := fileHeader.Open()
		if err != nil {
			uc.log.Errorf("failed to open file %d (%s): %v", i, fileHeader.Filename, err)
			continue
		}
		defer file.Close()

		// 保存到minio
		key := fmt.Sprintf("feedback/award/%s/%d/setted/%s", uid, feedbackAward.ID, fileHeader.Filename)
		err = uc.minioRepo.UploadFile(ctx, key, &file)
		if err != nil {
			uc.log.Errorf("failed to upload file %d (%s): %v", i, fileHeader.Filename, err)
			continue
		}

		uc.log.Infof("processing file %d: name=%s, size=%d bytes", i+1, fileHeader.Filename, fileHeader.Size)
		processedFiles = append(processedFiles, key)
	}

	/* if len(processedFiles) == 0 {
		return errors.New("no valid files to process")
	} */
	uc.log.Infof("successfully processed %d files: %v", len(processedFiles), processedFiles)

	// failed to process some files
	if len(processedFiles) != len(files) {
		uc.entClient.Award.DeleteOneID(feedbackAward.ID).Exec(ctx)
		for _, key := range processedFiles {
			uc.minioRepo.RemoveFile(ctx, key)
		}

		return errors.New("failed to process some files")
	}

	_, err = uc.entClient.Award.UpdateOneID(feedbackAward.ID).
		SetSettedFiles(processedFiles).
		Save(ctx)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Content-Type", "application/json")
	resObj := gabs.New()
	resObj.Set(feedbackAward.ID, "id")
	resObj.Set(now, "time")
	resObj.Set(processedFiles, "files")
	ctx.Response().Write(resObj.Bytes())

	return nil
}

func (uc *FeedbackUsecase) RealizeFeedbackAward(ctx kratosHttp.Context) error {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return errors.New("uid is empty")
	}

	req := ctx.Request()

	id := cast.ToUint64(req.FormValue("id"))
	now := time.Now().Local()

	awd, err := uc.entClient.Award.Get(ctx, id)
	if err != nil {
		return err
	}

	if awd.UserID != uid {
		return errors.New("you are not the owner of this award")
	}

	// 解析 multipart form 以处理多个文件
	err = req.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		return errors.New("failed to parse multipart form: " + err.Error())
	}

	// 获取所有上传的文件
	multipartForm := req.MultipartForm
	files, _ := multipartForm.File["files"]
	// 允许没有文件
	/* if !exists || len(files) == 0 {
		return errors.New("no files provided")
	} */

	uc.log.Infof("realize feedback award with %d files: id=%d, time=%s", 
		len(files), id, now)

	// 处理每个文件
	var processedFiles []string
	for i, fileHeader := range files {
		// 打开文件
		file, err := fileHeader.Open()
		if err != nil {
			uc.log.Errorf("failed to open file %d (%s): %v", i, fileHeader.Filename, err)
			continue
		}
		defer file.Close()

		key := fmt.Sprintf("feedback/award/%s/%d/realized/%s", uid, id, fileHeader.Filename)
		err = uc.minioRepo.UploadFile(ctx, key, &file)
		if err != nil {
			uc.log.Errorf("failed to upload file %d (%s): %v", i, fileHeader.Filename, err)
			continue
		}

		uc.log.Infof("processing file %d: name=%s, size=%d bytes", i+1, fileHeader.Filename, fileHeader.Size)
		processedFiles = append(processedFiles, key)
	}

	uc.log.Infof("successfully processed %d files: %v", len(processedFiles), processedFiles)

	// failed to process some files
	if len(processedFiles) != len(files) {
		for _, key := range processedFiles {
			uc.minioRepo.RemoveFile(ctx, key)
		}
		return fmt.Errorf("failed to process some files")
	}

	_, err = uc.entClient.Award.UpdateOneID(id).
		SetRealizedFiles(processedFiles).
		SetRealizedAt(now.Unix()).
		SetStatus(award.StatusRealized).
		Save(ctx)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Content-Type", "application/json")
	resObj := gabs.New()
	resObj.Set(id, "id")
	resObj.Set(now, "time")
	resObj.Set(processedFiles, "files")
	ctx.Response().Write(resObj.Bytes())

	return nil
}

func (uc *FeedbackUsecase) GetFeedbackAwards(ctx context.Context, req *stepApi.GetFeedbackAwardsRequest) (*stepApi.GetFeedbackAwardsReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	page := cast.ToInt(req.Page)
	pageSize := cast.ToInt(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	query := uc.entClient.Award.Query().Where(award.UserID(uid))
	if req.Status != "" {
		query = query.Where(award.StatusEQ(award.Status(req.Status)))
	}
	if req.SettedStartDate != "" && req.SettedEndDate != "" {
		startDate, err := time.Parse("2006-01-02 15:04:05", req.SettedStartDate + " 00:00:00")
		if err != nil {
			return nil, err
		}
		endDate, err := time.Parse("2006-01-02 15:04:05", req.SettedEndDate + " 23:59:59")
		if err != nil {
			return nil, err
		}
		query = query.Where(
			award.SettedAtGTE(startDate.Unix()),
			award.SettedAtLTE(endDate.Unix()),
		)
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	awds, err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order(ent.Desc(award.FieldSettedAt)).All(ctx)
	if err != nil {
		return nil, err
	}

	reply := &stepApi.GetFeedbackAwardsReply{
		Total: uint64(total),
	}
	for _, awd := range awds {
		fbAward := &stepApi.FeedbackAward{
			Id: awd.ID,
			Status: string(awd.Status),
			Description: awd.Description,
			TargetType: string(awd.TargetType),
			Scope: awd.Scope,
			Dimension: awd.Dimension,
			Threshold: awd.Threshold,
			SettedAt: awd.SettedAt,
			RealizedAt: awd.RealizedAt,
		}

		reply.Awards = append(reply.Awards, fbAward)
	}

	return reply, nil
}

func (uc *FeedbackUsecase) GetFeedbackAward(ctx context.Context, req *stepApi.GetFeedbackAwardRequest) (*stepApi.GetFeedbackAwardReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	awd, err := uc.entClient.Award.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if awd.UserID != uid {
		return nil, errors.New("you are not the owner of this award")
	}

	fbAward := &stepApi.FeedbackAward{
		Id: awd.ID,
		Status: string(awd.Status),
		Description: awd.Description,
		TargetType: string(awd.TargetType),
		Scope: awd.Scope,
		Dimension: awd.Dimension,
		Threshold: awd.Threshold,
		SettedAt: awd.SettedAt,
		RealizedAt: awd.RealizedAt,
		SettedFiles: awd.SettedFiles,
	}

	var assignedSettedFiles []string
	var assignedRealizedFiles []string
	for _, key := range awd.SettedFiles {
		presignedUrl, err := uc.minioRepo.GetDownloadPreSignedUrl(ctx, key, uc.minio_endpoint_remote)
		if err != nil {
			return nil, err
		}
		assignedSettedFiles = append(assignedSettedFiles, presignedUrl)
	}
	fbAward.SettedFiles = assignedSettedFiles
	for _, key := range awd.RealizedFiles {
		presignedUrl, err := uc.minioRepo.GetDownloadPreSignedUrl(ctx, key, uc.minio_endpoint_remote)
		if err != nil {
			return nil, err
		}
		assignedRealizedFiles = append(assignedRealizedFiles, presignedUrl)
	}
	fbAward.RealizedFiles = assignedRealizedFiles

	return &stepApi.GetFeedbackAwardReply{
		Award: fbAward,
	}, nil
}

func (uc *FeedbackUsecase) DeleteFeedbackAward(ctx context.Context, req *stepApi.DeleteFeedbackAwardRequest) (*stepApi.DeleteFeedbackAwardReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	awd, err := uc.entClient.Award.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if awd.UserID != uid {
		return nil, errors.New("you are not the owner of this award")
	}

	err = uc.entClient.Award.DeleteOneID(req.Id).Exec(ctx)
	if err != nil {
		return nil, err
	}

	for _, key := range awd.SettedFiles {
		uc.minioRepo.RemoveFile(ctx, key)
	}

	for _, key := range awd.RealizedFiles {
		uc.minioRepo.RemoveFile(ctx, key)
	}

	return &stepApi.DeleteFeedbackAwardReply{
		Id: req.Id,
	}, nil
}



