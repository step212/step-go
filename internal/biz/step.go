package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	stepApi "step/api/step/v1"

	"step/internal/data/ent"
	entStep "step/internal/data/ent/step"
	entTarget "step/internal/data/ent/target"
	"step/internal/objects"
	"step/internal/utils"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/spf13/cast"
)

type AsynqEnqueueRepo interface {
	EnqueueTargetCreate(ctx context.Context, targetID uint64) error
	EnqueueStepCreate(ctx context.Context, stepID uint64) error
	EnqueueStepComment(ctx context.Context, stepID uint64, commentType string) error
	EnqueueFeedbackPortraitChange(ctx context.Context, userID string, portraitChangeTypes []*objects.PortraitchangeType) error
}

type StepRepo interface {
	CreateTarget(ctx context.Context, req *stepApi.CreateTargetRequest) (*stepApi.CreateTargetReply, error)
	UpdateTarget(ctx context.Context, req *stepApi.UpdateTargetRequest) (*stepApi.UpdateTargetReply, error)
	GetTargets(ctx context.Context, req *stepApi.GetTargetsRequest) (*stepApi.GetTargetsReply, error)
	DeleteTarget(ctx context.Context, req *stepApi.DeleteTargetRequest) (*stepApi.DeleteTargetReply, error)
	DoneTarget(ctx context.Context, req *stepApi.DoneTargetRequest) (*stepApi.DoneTargetReply, error)
	GetTargetTree(ctx context.Context, req *stepApi.GetTargetTreeRequest) (*stepApi.GetTargetTreeReply, error)
	GetTarget(ctx context.Context, req *stepApi.GetTargetRequest) (*stepApi.GetTargetReply, error)
	UpdateTargetStep(ctx context.Context, req *stepApi.UpdateTargetStepRequest) (*stepApi.UpdateTargetStepReply, error)
	DeleteTargetStep(ctx context.Context, req *stepApi.DeleteTargetStepRequest) (*stepApi.DeleteTargetStepReply, error)
	AddTargetDirStep(ctx context.Context, req *stepApi.AddTargetDirStepRequest) (*stepApi.AddTargetDirStepReply, error)
	GetTargetDirStepChildren(ctx context.Context, req *stepApi.GetTargetDirStepChildrenRequest) (*stepApi.GetTargetDirStepChildrenReply, error)
	SetTargetStatusRecursively(ctx context.Context, targetID uint64, status entTarget.Status) error
	GetTargetByStepIDRecursively(ctx context.Context, stepID uint64, rootStepID uint64) (*ent.Target, error)
	GetTopTargetByTargetID(ctx context.Context, targetID uint64) (*ent.Target, error)
}

type StepUsecase struct {
	repo             StepRepo
	minioRepo        MinioRepo
	encryptRepo      EncryptRepo
	asynqEnqueueRepo AsynqEnqueueRepo
	entClient        *ent.Client
	log              *log.Helper
}

func NewStepUsecase(
	repo StepRepo,
	minioRepo MinioRepo,
	encryptRepo EncryptRepo,
	asynqEnqueueRepo AsynqEnqueueRepo,
	entClient *ent.Client,
	logger log.Logger,
) *StepUsecase {
	return &StepUsecase{
		repo:             repo,
		minioRepo:        minioRepo,
		encryptRepo:      encryptRepo,
		asynqEnqueueRepo: asynqEnqueueRepo,
		entClient:        entClient,
		log:              log.NewHelper(logger, log.WithMessageKey("stepUsecase")),
	}
}

func (uc *StepUsecase) Upload(ctx http.Context) error {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return errors.New("uid is empty")
	}

	req := ctx.Request()
	file, _, err := req.FormFile("file")
	if err != nil {
		return err
	}

	targetID := req.FormValue("targetID")
	parentID := req.FormValue("parentID")
	if parentID == "" && targetID == "" {
		return errors.New("parentID or targetID is required")
	}
	targetIDUint64 := cast.ToUint64(targetID)
	parentIDUint64 := cast.ToUint64(parentID)

	var target *ent.Target
	if targetIDUint64 != 0 {
		target, err = uc.entClient.Target.Query().
			Where(entTarget.ID(targetIDUint64)).
			First(ctx)
		if err != nil {
			return err
		}
	} else {
		target, err = uc.repo.GetTargetByStepIDRecursively(ctx, parentIDUint64, 0)
		if err != nil {
			return err
		}
	}

	if target.UserID != uid {
		return errors.New("not owner")
	}

	objectName := req.FormValue("objectName")
	if objectName == "" {
		return errors.New("objectName is required")
	}
	if targetIDUint64 != 0 {
		objectName = fmt.Sprintf("%s/target_%d/%s", uid, targetIDUint64, objectName)
	} else {
		objectName = fmt.Sprintf("%s/target_%d/dir_%d/%s", uid, target.ID, parentIDUint64, objectName)
	}

	fileType := req.FormValue("fileType")
	if fileType == "" {
		return errors.New("fileType is required")
	}

	isChallenge := req.FormValue("isChallenge")
	if isChallenge == "" {
		isChallenge = "false"
	}

	isChallengeBool, err := cast.ToBoolE(isChallenge)
	if err != nil {
		return err
	}

	title := req.FormValue("title")
	description := req.FormValue("description")

	var stepTime int64
	stepTimeStr := req.FormValue("stepTime")
	if stepTimeStr != "" {
		stepTime = cast.ToInt64(stepTimeStr)
	} else {
		stepTime = time.Now().Local().Unix()
	}

	err = uc.minioRepo.UploadFile(ctx, objectName, &file)
	if err != nil {
		return err
	}

	stepCreate := uc.entClient.Step.Create()
	if parentIDUint64 != 0 {
		stepCreate.SetParentID(parentIDUint64)
	}
	// 无论是否为目录性质的目标，都设置目标ID，为了方便统计
	stepCreate.SetTargetID(target.ID)

	step, err := stepCreate.
		SetIsChallenge(isChallengeBool).
		SetTitle(title).
		SetDescription(description).
		SetType(entStep.Type(fileType)).
		SetObjectName(objectName).
		SetCreatedAt(stepTime).
		Save(ctx)
	if err != nil {
		return err
	}

	var status entTarget.Status
	if isChallengeBool {
		status = entTarget.StatusStepHard
	} else {
		status = entTarget.StatusStep
	}

	err = uc.repo.SetTargetStatusRecursively(ctx, target.ID, status)
	if err != nil {
		return err
	}

	presignedUrl, err := uc.minioRepo.GetDownloadPreSignedUrl(ctx, objectName, uc.minioRepo.GetRemoteEndpoint(ctx))
	if err != nil {
		return err
	}

	response := ctx.Response()
	response.Header().Set("Content-Type", "application/json")
	resObj := gabs.New()
	resObj.Set(step.ID, "id")
	resObj.Set(step.Title, "title")
	resObj.Set(step.Description, "description")
	resObj.Set(step.Type, "type")
	resObj.Set(step.IsChallenge, "isChallenge")
	resObj.Set(objectName, "objectName")
	resObj.Set(presignedUrl, "presignedUrl")
	resObj.Set(step.CreatedAt, "createdAt")
	response.Write(resObj.Bytes())

	err = uc.asynqEnqueueRepo.EnqueueStepCreate(ctx, step.ID)
	if err != nil {
		uc.log.Errorf("EnqueueStepCreate error: %v", err)
		//return err
	}

	return nil
}

func (uc *StepUsecase) CreateTarget(ctx context.Context, req *stepApi.CreateTargetRequest) (*stepApi.CreateTargetReply, error) {
	reply, err := uc.repo.CreateTarget(ctx, req)
	if err != nil {
		return nil, err
	}

	err = uc.asynqEnqueueRepo.EnqueueTargetCreate(ctx, reply.Id)
	if err != nil {
		uc.log.Errorf("EnqueueTargetCreate error: %v", err)
		//return err
	}

	return reply, nil
}

func (uc *StepUsecase) UpdateTarget(ctx context.Context, req *stepApi.UpdateTargetRequest) (*stepApi.UpdateTargetReply, error) {
	return uc.repo.UpdateTarget(ctx, req)
}

func (uc *StepUsecase) GetTargets(ctx context.Context, req *stepApi.GetTargetsRequest) (*stepApi.GetTargetsReply, error) {
	return uc.repo.GetTargets(ctx, req)
}

func (uc *StepUsecase) DeleteTarget(ctx context.Context, req *stepApi.DeleteTargetRequest) (*stepApi.DeleteTargetReply, error) {
	return uc.repo.DeleteTarget(ctx, req)
}

func (uc *StepUsecase) DoneTarget(ctx context.Context, req *stepApi.DoneTargetRequest) (*stepApi.DoneTargetReply, error) {
	return uc.repo.DoneTarget(ctx, req)
}

func (uc *StepUsecase) GetTarget(ctx context.Context, req *stepApi.GetTargetRequest) (*stepApi.GetTargetReply, error) {
	return uc.repo.GetTarget(ctx, req)
}

func (uc *StepUsecase) GetTargetTree(ctx context.Context, req *stepApi.GetTargetTreeRequest) (*stepApi.GetTargetTreeReply, error) {
	return uc.repo.GetTargetTree(ctx, req)
}

func (uc *StepUsecase) AddTargetDirStep(ctx context.Context, req *stepApi.AddTargetDirStepRequest) (*stepApi.AddTargetDirStepReply, error) {
	return uc.repo.AddTargetDirStep(ctx, req)
}

func (uc *StepUsecase) GetTargetDirStepChildren(ctx context.Context, req *stepApi.GetTargetDirStepChildrenRequest) (*stepApi.GetTargetDirStepChildrenReply, error) {
	return uc.repo.GetTargetDirStepChildren(ctx, req)
}

func (uc *StepUsecase) UpdateTargetStep(ctx context.Context, req *stepApi.UpdateTargetStepRequest) (*stepApi.UpdateTargetStepReply, error) {
	return uc.repo.UpdateTargetStep(ctx, req)
}

func (uc *StepUsecase) DeleteTargetStep(ctx context.Context, req *stepApi.DeleteTargetStepRequest) (*stepApi.DeleteTargetStepReply, error) {
	return uc.repo.DeleteTargetStep(ctx, req)
}

func (uc *StepUsecase) Encrypt(ctx context.Context, req *stepApi.EncryptRequest) (*stepApi.EncryptReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := uc.repo.GetTargetByStepIDRecursively(ctx, req.Id, 0)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	encrypted, err := uc.encryptRepo.Encrypt(ctx, req.Id, req.Data)
	if err != nil {
		return nil, err
	}
	return &stepApi.EncryptReply{Data: encrypted}, nil
}
