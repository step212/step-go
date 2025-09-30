package data

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	stepApi "step/api/step/v1"
	"step/internal/biz"
	"step/internal/data/ent"
	entStep "step/internal/data/ent/step"

	"github.com/go-kratos/kratos/v2/log"
)

type stepNoauthRepo struct {
	data        *Data
	log         *log.Helper
	minioRepo   biz.MinioRepo
	encryptRepo biz.EncryptRepo
}

func NewStepNoauthRepo(data *Data, logger log.Logger, minioRepo biz.MinioRepo, encryptRepo biz.EncryptRepo) biz.StepNoauthRepo {
	return &stepNoauthRepo{
		data:        data,
		log:         log.NewHelper(logger, log.WithMessageKey("stepNoauthRepo")),
		minioRepo:   minioRepo,
		encryptRepo: encryptRepo,
	}
}

func (r *stepNoauthRepo) findTopStepToGetTarget(ctx context.Context, stepId uint64) (*ent.Target, error) {
	step, err := r.data.ent_client.Step.Query().Where(entStep.ID(stepId)).WithParent().WithTarget().First(ctx)
	if err != nil {
		return nil, err
	}

	if step.Edges.Target != nil {
		return step.Edges.Target, nil
	}

	if step.Edges.Parent == nil {
		return nil, errors.New("can't fine target of this step")
	}

	return r.findTopStepToGetTarget(ctx, step.Edges.Parent.ID)
}

func (r *stepNoauthRepo) getStepForApi(ctx context.Context, step *ent.Step, shareTo string) (*stepApi.Step, error) {
	shareToRoles := make(map[string]bool)
	if shareTo != "" {
		for _, role := range strings.Split(shareTo, ",") {
			shareToRoles[strings.TrimSpace(role)] = true
		}
	}

	var teacherJsonComment, parentJsonComment, friendJsonComment []byte
	var err error
	if shareToRoles["teacher"] && step.TeacherComment != nil {
		teacherJsonComment, err = json.Marshal(step.TeacherComment)
		if err != nil {
			return nil, err
		}
	}
	if shareToRoles["parent"] && step.ParentComment != nil {
		parentJsonComment, err = json.Marshal(step.ParentComment)
		if err != nil {
			return nil, err
		}
	}
	if shareToRoles["friend"] && step.FriendComment != nil {
		friendJsonComment, err = json.Marshal(step.FriendComment)
		if err != nil {
			return nil, err
		}
	}

	var presignedUrl string
	if step.Type != entStep.TypeDir {
		presignedUrl, err = r.minioRepo.GetDownloadPreSignedUrl(ctx, step.ObjectName, r.data.minio_endpoint_remote)
		if err != nil {
			return nil, err
		}
	}

	stepForApi := &stepApi.Step{
		Id:             step.ID,
		Type:           step.Type.String(),
		Title:          step.Title,
		Description:    step.Description,
		IsChallenge:    step.IsChallenge,
		ObjectName:     step.ObjectName,
		PresignedUrl:   presignedUrl,
		TeacherComment: string(teacherJsonComment),
		ParentComment:  string(parentJsonComment),
		FriendComment:  string(friendJsonComment),
		CreatedAt:      step.CreatedAt,
	}

	return stepForApi, nil
}

func (r *stepNoauthRepo) GetNoauthStep(ctx context.Context, req *stepApi.GetNoauthStepRequest) (*stepApi.GetNoauthStepReply, error) {
	decryptedShareTo, err := r.encryptRepo.Decrypt(ctx, req.Id, req.ShareTo)
	if err != nil {
		return nil, err
	}

	target, err := r.findTopStepToGetTarget(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	step, err := r.data.ent_client.Step.Query().Where(entStep.ID(req.Id)).First(ctx)
	if err != nil {
		return nil, err
	}

	stepForApi, err := r.getStepForApi(ctx, step, decryptedShareTo)
	if err != nil {
		return nil, err
	}

	childrenApi := make([]*stepApi.Step, 0)
	if step.Type == entStep.TypeDir {
		children, err := r.data.ent_client.Step.Query().Where(entStep.ParentID(step.ID)).All(ctx)
		if err != nil {
			return nil, err
		}

		for _, child := range children {
			childApi, err := r.getStepForApi(ctx, child, decryptedShareTo)
			if err != nil {
				return nil, err
			}
			childrenApi = append(childrenApi, childApi)
		}
	}

	return &stepApi.GetNoauthStepReply{
		TargetTitle:       target.Title,
		TargetDescription: target.Description,
		Step:              stepForApi,
		Children:          childrenApi,
	}, nil
}

func (r *stepNoauthRepo) SetCommentForStep(ctx context.Context, req *stepApi.SetCommentForStepRequest) (*stepApi.SetCommentForStepReply, error) {
	if req.Type != "teacher" && req.Type != "parent" && req.Type != "friend" {
		return nil, errors.New("invalid comment type")
	}

	commentMap := make(map[string]any)
	err := json.Unmarshal([]byte(req.Comment), &commentMap)
	if err != nil {
		return nil, err
	}

	step, err := r.data.ent_client.Step.Query().Where(entStep.ID(req.Id)).First(ctx)
	if err != nil {
		return nil, err
	}

	/* // step只能在24小时内被评论，且只能被评论一次
	now := time.Now().Local()
	stepCreatedAt := time.Unix(step.CreatedAt, 0)
	if now.After(stepCreatedAt.Add(24 * time.Hour)) {
		return nil, errors.New("step can't be commented after 24 hours")
	} */

	if req.Type == "teacher" && step.TeacherComment != nil {
		return nil, errors.New("step already has teacher comment")
	} else if req.Type == "parent" && step.ParentComment != nil {
		return nil, errors.New("step already has parent comment")
	} else if req.Type == "friend" && step.FriendComment != nil {
		return nil, errors.New("step already has friend comment")
	}

	updateStep := step.Update().Where(entStep.ID(req.Id))

	if req.Type == "teacher" {
		updateStep.SetTeacherComment(commentMap)
	} else if req.Type == "parent" {
		updateStep.SetParentComment(commentMap)
	} else if req.Type == "friend" {
		updateStep.SetFriendComment(commentMap)
	}

	_, err = updateStep.Save(ctx)
	if err != nil {
		return nil, err
	}

	return &stepApi.SetCommentForStepReply{
		Id: req.Id,
	}, nil
}
