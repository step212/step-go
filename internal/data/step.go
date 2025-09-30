package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"step/internal/biz"
	"step/internal/data/ent"
	entStep "step/internal/data/ent/step"
	entTarget "step/internal/data/ent/target"
	"step/internal/utils"
	"time"

	stepApi "step/api/step/v1"

	"github.com/go-kratos/kratos/v2/log"
)

type stepRepo struct {
	data      *Data
	log       *log.Helper
	minioRepo biz.MinioRepo
}

func NewStepRepo(data *Data, logger log.Logger, minioRepo biz.MinioRepo) biz.StepRepo {
	return &stepRepo{
		data:      data,
		log:       log.NewHelper(logger, log.WithMessageKey("stepRepo")),
		minioRepo: minioRepo,
	}
}

func (r *stepRepo) CreateTarget(ctx context.Context, req *stepApi.CreateTargetRequest) (*stepApi.CreateTargetReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	now := time.Now().Local().Unix()

	createTarget := r.data.ent_client.Target.Create().
		SetUserID(uid).
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetCreatedAt(now).
		SetStatus(entTarget.StatusInit)

	if req.Type != "" {
		createTarget.SetType(req.Type)
	}

	if req.ParentId != 0 {
		parentTarget, err := r.data.ent_client.Target.Get(ctx, req.ParentId)
		if err != nil {
			return nil, err
		}

		if parentTarget.UserID != uid {
			return nil, errors.New("not owner")
		}

		createTarget.SetLayer(parentTarget.Layer + 1)
		createTarget.SetParentID(req.ParentId)
	} else {
		createTarget.SetLayer(0)
	}

	target, err := createTarget.Save(ctx)
	if err != nil {
		return nil, err
	}

	return &stepApi.CreateTargetReply{
		Id:    target.ID,
		Layer: target.Layer,
	}, nil
}

func (r *stepRepo) UpdateTarget(ctx context.Context, req *stepApi.UpdateTargetRequest) (*stepApi.UpdateTargetReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := r.data.ent_client.Target.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	_, err = target.Update().
		SetTitle(req.Title).
		SetDescription(req.Description).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return &stepApi.UpdateTargetReply{
		Id: target.ID,
	}, nil
}

func (r *stepRepo) GetTargets(ctx context.Context, req *stepApi.GetTargetsRequest) (*stepApi.GetTargetsReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	if req.ParentId == 0 {
		targets, err := r.data.ent_client.Target.Query().
			Where(entTarget.UserID(uid)).
			Where(entTarget.Or(entTarget.ParentIDIsNil(), entTarget.ParentID(0))).
			Order(ent.Desc(entTarget.FieldID)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		targetsApi := make([]*stepApi.Target, len(targets))
		for i, target := range targets {
			targetsApi[i] = &stepApi.Target{
				Id:          target.ID,
				Title:       target.Title,
				Description: target.Description,
				Type:        target.Type,
				CreatedAt:   target.CreatedAt,
				StartAt:     target.StartAt,
				ChallengeAt: target.ChallengeAt,
				DoneAt:      target.DoneAt,
				Layer:       target.Layer,
				Status:      target.Status.String(),
			}
		}
		return &stepApi.GetTargetsReply{
			Targets: targetsApi,
		}, nil
	} else {
		target, err := r.data.ent_client.Target.Get(ctx, req.ParentId)
		if err != nil {
			return nil, err
		}

		if target.UserID != uid {
			return nil, errors.New("not owner")
		}

		targets, err := target.QueryChildren().Order(ent.Desc(entTarget.FieldID)).All(ctx)
		if err != nil {
			return nil, err
		}

		targetsApi := make([]*stepApi.Target, len(targets))
		for i, target := range targets {
			targetsApi[i] = &stepApi.Target{
				Id:           target.ID,
				Title:        target.Title,
				Description:  target.Description,
				Type:         target.Type,
				CreatedAt:    target.CreatedAt,
				StartAt:      target.StartAt,
				ChallengeAt:  target.ChallengeAt,
				DoneAt:       target.DoneAt,
				Layer:        target.Layer,
				Status:       target.Status.String(),
				TargetParent: target.ParentID,
			}
		}

		return &stepApi.GetTargetsReply{
			Targets: targetsApi,
		}, nil
	}
}

func (r *stepRepo) DeleteTarget(ctx context.Context, req *stepApi.DeleteTargetRequest) (*stepApi.DeleteTargetReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := r.data.ent_client.Target.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	err = r.data.ent_client.Target.DeleteOneID(target.ID).Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 当前只删除本目标，子目标及其积累暂不删除

	return &stepApi.DeleteTargetReply{
		Id: target.ID,
	}, nil
}

func (r *stepRepo) DoneTarget(ctx context.Context, req *stepApi.DoneTargetRequest) (*stepApi.DoneTargetReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := r.data.ent_client.Target.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	err = r.SetTargetStatusRecursively(ctx, target.ID, entTarget.StatusDone)
	if err != nil {
		return nil, err
	}

	return &stepApi.DoneTargetReply{
		Id: target.ID,
	}, nil
}

func (r *stepRepo) findRootTarget(ctx context.Context, target *ent.Target) (*ent.Target, error) {
	if target.Layer == 0 {
		return target, nil
	}

	if target.ParentID == 0 {
		return nil, fmt.Errorf("target %d with layer %d has no parent", target.ID, target.Layer)
	}

	parentTarget, err := r.data.ent_client.Target.Get(ctx, target.ParentID)
	if err != nil {
		return nil, err
	}

	if parentTarget.Layer == 0 {
		return parentTarget, nil
	}

	rootTarget, err := r.findRootTarget(ctx, parentTarget)
	if err != nil {
		return nil, err
	}
	return rootTarget, nil
}

func (r *stepRepo) getChildrenTargets(ctx context.Context, target *stepApi.Target) error {
	childrenTargets, err := r.data.ent_client.Target.Query().
		Where(entTarget.ParentID(target.Id)).
		All(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return err
	}

	childrenTargetsApi := make([]*stepApi.Target, len(childrenTargets))
	for i, target := range childrenTargets {
		childrenTargetsApi[i] = &stepApi.Target{
			Id:           target.ID,
			Title:        target.Title,
			Description:  target.Description,
			Type:         target.Type,
			CreatedAt:    target.CreatedAt,
			StartAt:      target.StartAt,
			ChallengeAt:  target.ChallengeAt,
			DoneAt:       target.DoneAt,
			Layer:        target.Layer,
			Status:       target.Status.String(),
			TargetParent: target.ParentID,
		}

		err = r.getChildrenTargets(ctx, childrenTargetsApi[i])
		if err != nil {
			return err
		}
	}
	target.Children = childrenTargetsApi

	return nil
}

func (r *stepRepo) GetTargetTree(ctx context.Context, req *stepApi.GetTargetTreeRequest) (*stepApi.GetTargetTreeReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := r.data.ent_client.Target.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	rootTarget, err := r.findRootTarget(ctx, target)
	if err != nil {
		return nil, err
	}

	rootTargetApi := &stepApi.Target{
		Id:          rootTarget.ID,
		Title:       rootTarget.Title,
		Description: rootTarget.Description,
		Type:        rootTarget.Type,
		CreatedAt:   rootTarget.CreatedAt,
		StartAt:     rootTarget.StartAt,
		ChallengeAt: rootTarget.ChallengeAt,
		DoneAt:      rootTarget.DoneAt,
		Layer:       rootTarget.Layer,
		Status:      rootTarget.Status.String(),
	}

	err = r.getChildrenTargets(ctx, rootTargetApi)
	if err != nil {
		return nil, err
	}

	return &stepApi.GetTargetTreeReply{
		RootTarget: rootTargetApi,
	}, nil
}

func (r *stepRepo) GetTarget(ctx context.Context, req *stepApi.GetTargetRequest) (*stepApi.GetTargetReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := r.data.ent_client.Target.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	reply := &stepApi.GetTargetReply{
		Target: &stepApi.Target{
			Id:           target.ID,
			Title:        target.Title,
			Description:  target.Description,
			Type:         target.Type,
			CreatedAt:    target.CreatedAt,
			StartAt:      target.StartAt,
			ChallengeAt:  target.ChallengeAt,
			DoneAt:       target.DoneAt,
			Layer:        target.Layer,
			Status:       target.Status.String(),
			TargetParent: target.ParentID,
		},
	}

	if req.WithSteps {
		page := req.Page
		if page == 0 {
			page = 1
		}

		pageSize := req.PageSize
		if pageSize == 0 {
			pageSize = 10
		}

		query := target.QuerySteps()

		total, err := query.Count(ctx)
		if err != nil {
			return nil, err
		}
		reply.Total = uint64(total)

		steps, err := query.Order(ent.Desc(entStep.FieldID)).Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).All(ctx)
		if err != nil {
			return nil, err
		}

		stepsApi := make([]*stepApi.Step, len(steps))
		for i, step := range steps {
			jsonTeacherComment, err := json.Marshal(step.TeacherComment)
			if err != nil {
				return nil, err
			}

			jsonParentComment, err := json.Marshal(step.ParentComment)
			if err != nil {
				return nil, err
			}

			jsonFriendComment, err := json.Marshal(step.FriendComment)
			if err != nil {
				return nil, err
			}

			stepsApi[i] = &stepApi.Step{
				Id:             step.ID,
				Title:          step.Title,
				Description:    step.Description,
				IsChallenge:    step.IsChallenge,
				Type:           step.Type.String(),
				ObjectName:     step.ObjectName,
				CreatedAt:      step.CreatedAt,
				RefTargetId:    step.RefTargetID,
				TeacherComment: string(jsonTeacherComment),
				ParentComment:  string(jsonParentComment),
				FriendComment:  string(jsonFriendComment),
			}

			if step.Type != entStep.TypeDir {
				presignedUrl, err := r.minioRepo.GetDownloadPreSignedUrl(ctx, step.ObjectName, r.data.minio_endpoint_remote)
				if err != nil {
					return nil, err
				}
				stepsApi[i].PresignedUrl = presignedUrl
			}
		}
		reply.Steps = stepsApi
	}

	return reply, nil
}

func (r *stepRepo) AddTargetDirStep(ctx context.Context, req *stepApi.AddTargetDirStepRequest) (*stepApi.AddTargetDirStepReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := r.data.ent_client.Target.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	step, err := r.data.ent_client.Step.Create().
		SetTitle(req.Title).
		SetType(entStep.TypeDir).
		SetCreatedAt(time.Now().Local().Unix()).
		SetRefTargetID(req.Id).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return &stepApi.AddTargetDirStepReply{
		Id: step.ID,
	}, nil
}

func (r *stepRepo) GetTargetDirStepChildren(ctx context.Context, req *stepApi.GetTargetDirStepChildrenRequest) (*stepApi.GetTargetDirStepChildrenReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := r.GetTargetByStepIDRecursively(ctx, req.Id, 0)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	page := req.Page
	if page == 0 {
		page = 1
	}

	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 10
	}

	query := r.data.ent_client.Step.Query().
		Where(entStep.ParentIDEQ(req.Id))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	steps, err := query.Order(ent.Desc(entStep.FieldID)).Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).All(ctx)
	if err != nil {
		return nil, err
	}

	stepsApi := make([]*stepApi.Step, len(steps))
	for i, step := range steps {
		jsonTeacherComment, err := json.Marshal(step.TeacherComment)
		if err != nil {
			return nil, err
		}

		jsonParentComment, err := json.Marshal(step.ParentComment)
		if err != nil {
			return nil, err
		}

		jsonFriendComment, err := json.Marshal(step.FriendComment)
		if err != nil {
			return nil, err
		}

		stepsApi[i] = &stepApi.Step{
			Id:             step.ID,
			Title:          step.Title,
			Description:    step.Description,
			IsChallenge:    step.IsChallenge,
			Type:           step.Type.String(),
			ObjectName:     step.ObjectName,
			CreatedAt:      step.CreatedAt,
			RefTargetId:    step.RefTargetID,
			TeacherComment: string(jsonTeacherComment),
			ParentComment:  string(jsonParentComment),
			FriendComment:  string(jsonFriendComment),
		}

		presignedUrl, err := r.minioRepo.GetDownloadPreSignedUrl(ctx, step.ObjectName, r.data.minio_endpoint_remote)
		if err != nil {
			return nil, err
		}
		stepsApi[i].PresignedUrl = presignedUrl
	}

	return &stepApi.GetTargetDirStepChildrenReply{
		Total: uint64(total),
		Steps: stepsApi,
	}, nil
}

func (r *stepRepo) UpdateTargetStep(ctx context.Context, req *stepApi.UpdateTargetStepRequest) (*stepApi.UpdateTargetStepReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := r.GetTargetByStepIDRecursively(ctx, req.Id, 0)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	updateStep := r.data.ent_client.Step.UpdateOneID(req.Id)

	if req.Title != "" {
		updateStep.SetTitle(req.Title)
	}

	if req.Description != "" {
		updateStep.SetDescription(req.Description)
	}

	if req.IsChallenge != nil {
		updateStep.SetIsChallenge(req.IsChallenge.GetValue())
	}

	if req.Type != "" {
		updateStep.SetType(entStep.Type(req.Type))
	}

	_, err = updateStep.Save(ctx)
	if err != nil {
		return nil, err
	}

	return &stepApi.UpdateTargetStepReply{
		Id: req.Id,
	}, nil
}

func (r *stepRepo) DeleteTargetStep(ctx context.Context, req *stepApi.DeleteTargetStepRequest) (*stepApi.DeleteTargetStepReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	target, err := r.GetTargetByStepIDRecursively(ctx, req.Id, 0)
	if err != nil {
		return nil, err
	}

	if target.UserID != uid {
		return nil, errors.New("not owner")
	}

	step, err := r.data.ent_client.Step.Query().
		Where(entStep.ID(req.Id)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	if step.ObjectName != "" {
		err = r.minioRepo.RemoveFile(ctx, step.ObjectName)
		if err != nil {
			return nil, err
		}
	}

	err = r.data.ent_client.Step.DeleteOneID(step.ID).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &stepApi.DeleteTargetStepReply{
		Id: step.ID,
	}, nil
}

func (r *stepRepo) SetTargetStatusRecursively(ctx context.Context, targetID uint64, status entTarget.Status) error {
	switch status {
	case entTarget.StatusStep:
		target, err := r.data.ent_client.Target.Query().
			Where(entTarget.ID(targetID)).
			WithParent().
			First(ctx)
		if err != nil {
			return err
		}

		if target.Status == entTarget.StatusInit {
			_, err = target.Update().
				SetStatus(status).
				SetStartAt(time.Now().Local().Unix()).
				Save(ctx)
			if err != nil {
				return err
			}

			// 递归设置父目标状态
			if target.Edges.Parent != nil {
				err = r.SetTargetStatusRecursively(ctx, target.Edges.Parent.ID, status)
				if err != nil {
					return err
				}
			}
		}

		return nil

	case entTarget.StatusStepHard:
		target, err := r.data.ent_client.Target.Query().
			Where(entTarget.ID(targetID)).
			WithParent().
			First(ctx)
		if err != nil {
			return err
		}

		if target.Status == entTarget.StatusInit || target.Status == entTarget.StatusStep {
			_, err = target.Update().
				SetStatus(status).
				SetChallengeAt(time.Now().Local().Unix()).
				Save(ctx)
			if err != nil {
				return err
			}

			// 递归设置父目标状态
			if target.Edges.Parent != nil {
				err = r.SetTargetStatusRecursively(ctx, target.Edges.Parent.ID, status)
				if err != nil {
					return err
				}
			}
		}

		return nil

	case entTarget.StatusDone:
		target, err := r.data.ent_client.Target.Query().
			Where(entTarget.ID(targetID)).
			WithParent().
			First(ctx)
		if err != nil {
			return err
		}

		if target.Status != entTarget.StatusDone {
			_, err = target.Update().
				SetStatus(status).
				SetDoneAt(time.Now().Local().Unix()).
				Save(ctx)
			if err != nil {
				return err
			}

			if target.Edges.Parent != nil {
				existNonDoneChild, err := r.data.ent_client.Target.Query().
					Where(entTarget.ParentID(target.Edges.Parent.ID), entTarget.StatusNEQ(entTarget.StatusDone)).
					Exist(ctx)
				if err != nil {
					return err
				}

				if !existNonDoneChild {
					err = r.SetTargetStatusRecursively(ctx, target.Edges.Parent.ID, entTarget.StatusDone)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	default:
		return fmt.Errorf("invalid status: %s", status)
	}
}

func (r *stepRepo) GetTargetByStepIDRecursively(ctx context.Context, stepID uint64, rootStepID uint64) (*ent.Target, error) {
	if rootStepID == 0 {
		rootStepID = stepID
	}

	step, err := r.data.ent_client.Step.Query().
		Where(entStep.ID(stepID)).
		WithTarget().
		WithParent().
		First(ctx)
	if err != nil {
		return nil, err
	}

	if step.Edges.Target != nil {
		return step.Edges.Target, nil
	}

	// there is no associated target and no parent step
	if step.Edges.Parent == nil {
		return nil, fmt.Errorf("step %d has no associated target", rootStepID)
	} else {
		return r.GetTargetByStepIDRecursively(ctx, step.Edges.Parent.ID, rootStepID)
	}
}

func (r *stepRepo) GetTopTargetByTargetID(ctx context.Context, targetID uint64) (*ent.Target, error) {
	target, err := r.data.ent_client.Target.Query().
		Where(entTarget.ID(targetID)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	if target.Layer == 0 || target.ParentID == 0 {
		return target, nil
	}

	return r.GetTopTargetByTargetID(ctx, target.ParentID)
}
