package data

import (
	"context"
	"encoding/json"
	"fmt"
	"step/internal/biz"
	"step/internal/data/ent"
	"step/internal/data/ent/portrait"
	"step/internal/data/ent/step"
	"step/internal/data/ent/steprate"
	"step/internal/data/ent/target"
	"step/internal/objects"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
)

type statisticsRepo struct {
	data     *Data
	log      *log.Helper
	stepRepo biz.StepRepo
}

// 每次评价都check一次基本数据是否存在着；如果不存在，则创建，默认值为0
func NewStatisticsRepo(data *Data, logger log.Logger, stepRepo biz.StepRepo) biz.StatisticsRepo {
	return &statisticsRepo{
		data:     data,
		log:      log.NewHelper(logger, log.WithMessageKey("statisticsRepo")),
		stepRepo: stepRepo,
	}
}

func (r statisticsRepo) CheckStatistics(ctx context.Context, userID string) error {
	// portrait基本要求：勇敢、果断、耐心、毅力
	_, err := r.data.ent_client.Portrait.Query().
		Where(portrait.UserID(userID), portrait.DimensionEQ(portrait.DimensionBasic)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		}

		_, err = r.data.ent_client.Portrait.Create().
			SetUserID(userID).
			SetDimension(portrait.DimensionBasic).
			SetValue(map[string]any{
				objects.PortraitBasicBravery:      0,
				objects.PortraitBasicDecisiveness: 0,
				objects.PortraitBasicPatience:     0,
				objects.PortraitBasicPerseverance: 0,
			}).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	// portrait自律性：打卡频率、持续性、目标达成度、挑战心态（迎难而上，不惧困难）
	_, err = r.data.ent_client.Portrait.Query().
		Where(portrait.UserID(userID), portrait.DimensionEQ(portrait.DimensionSelfDiscipline)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		}

		_, err = r.data.ent_client.Portrait.Create().
			SetUserID(userID).
			SetDimension(portrait.DimensionSelfDiscipline).
			SetValue(map[string]any{
				objects.PortraitSelfDisciplineCheckinFrequency:  0,
				objects.PortraitSelfDisciplineConsistency:       0,
				objects.PortraitSelfDisciplineGoalAchievement:   0,
				objects.PortraitSelfDisciplineChallengeAttitude: 0,
			}).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	// portrait目标设定与执行能力：目标合理性、目标明确性、目标达成度、调整能力
	_, err = r.data.ent_client.Portrait.Query().
		Where(portrait.UserID(userID), portrait.DimensionEQ(portrait.DimensionTargetAndExecution)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		}

		_, err = r.data.ent_client.Portrait.Create().
			SetUserID(userID).
			SetDimension(portrait.DimensionTargetAndExecution).
			SetValue(map[string]any{
				objects.PortraitTargetAndExecutionGoalReasonableness: 0,
				objects.PortraitTargetAndExecutionGoalClarity:        0,
				objects.PortraitTargetAndExecutionGoalAchievement:    0,
				objects.PortraitTargetAndExecutionAdjustmentAbility:  0,
			}).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	// portrait学习与成长能力：反思与改进、创新方法、基础牢靠、技能提升、挑战心态（拔尖能力）
	_, err = r.data.ent_client.Portrait.Query().
		Where(portrait.UserID(userID), portrait.DimensionEQ(portrait.DimensionLearningAndGrowth)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		}

		_, err = r.data.ent_client.Portrait.Create().
			SetUserID(userID).
			SetDimension(portrait.DimensionLearningAndGrowth).
			SetValue(map[string]any{
				objects.PortraitLearningAndGrowthReflectionAndImprovement: 0,
				objects.PortraitLearningAndGrowthInnovativeMethod:         0,
				objects.PortraitLearningAndGrowthBasicSolid:               0,
				objects.PortraitLearningAndGrowthSkillImprovement:         0,
				objects.PortraitLearningAndGrowthChallengeAttitude:        0,
			}).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// 每次目标的创建，都需要勇气，所以需要增加勇气值1
func (r statisticsRepo) HandleTargetCreate(ctx context.Context, task *asynq.Task) (uid string, portraitChangeTypes []*objects.PortraitchangeType, err error) {
	var payload objects.TargetCreatePayload
	err = json.Unmarshal(task.Payload(), &payload)
	if err != nil {
		return "", nil, err
	}

	t, err := r.data.ent_client.Target.Query().
		Where(target.ID(payload.TargetID)).
		Only(ctx)
	if err != nil {
		return "", nil, err
	}

	err = r.CheckStatistics(ctx, t.UserID)
	if err != nil {
		return "", nil, err
	}

	// 获取当前的 portrait 数据
	p, err := r.data.ent_client.Portrait.Query().
		Where(portrait.UserID(t.UserID), portrait.DimensionEQ(portrait.DimensionBasic)).
		Only(ctx)
	if err != nil {
		return "", nil, err
	}

	// 读取当前值并递增勇气值1
	currentValues := p.Value
	if braveryVal, ok := currentValues[objects.PortraitBasicBravery]; ok {
		braveryValInt := cast.ToInt(braveryVal)
		currentValues[objects.PortraitBasicBravery] = braveryValInt + 1
	} else {
		currentValues[objects.PortraitBasicBravery] = 1
	}

	// 更新整个值
	_, err = r.data.ent_client.Portrait.UpdateOneID(p.ID).
		SetValue(currentValues).
		Save(ctx)
	if err != nil {
		return "", nil, err
	}

	return t.UserID, []*objects.PortraitchangeType{
		{Type: "portrait", Scope: []string{portrait.DimensionBasic.String()}},
	}, nil
}

// 负面评价：评价在0-10之间，但是允许存在负面评价，所以需要统一-5，即5分为中间值，低于5分为负面评价。
// ● 每次积累创建，如果是非挑战性的，增加耐心值1；如果是挑战性的，增加毅力值1。
// ● 每次积累创建，查询是否是第一次创建，如果是，根据目标设定时间和第一次打卡时间算出需要增加的果断值（0-10）。
// ● 每次积累创建，需要同步在step_rates表中新建打卡记录，weighted_value设置为默认值0
// ● 每次积累创建，算出打卡频率指标，每周、每月几次
// ● 每次积累创建，算出连续性指标，连续几天就增加几，最大10；超过3天没有打卡，则为负面评价，需要减少连续性值
func (r statisticsRepo) HandleStepCreate(ctx context.Context, task *asynq.Task) (uid string, portraitChangeTypes []*objects.PortraitchangeType, err error) {
	var payload objects.StepCreatePayload
	err = json.Unmarshal(task.Payload(), &payload)
	if err != nil {
		return "", nil, err
	}

	s, err := r.data.ent_client.Step.Query().
		Where(step.ID(payload.StepID)).
		WithTarget().
		Only(ctx)
	if err != nil {
		return "", nil, err
	}

	if s.Type == step.TypeDir {
		return "", nil, fmt.Errorf("no action for dir step")
	}

	/* t, err := r.stepRepo.GetTargetByStepIDRecursively(ctx, payload.StepID, 0)
	if err != nil {
		return "", nil, err
	} */
	t := s.Edges.Target

	topTarget, err := r.stepRepo.GetTopTargetByTargetID(ctx, t.ID)
	if err != nil {
		return "", nil, err
	}

	err = r.CheckStatistics(ctx, t.UserID)
	if err != nil {
		return "", nil, err
	}

	portraitBasic, err := r.data.ent_client.Portrait.Query().
		Where(portrait.UserID(t.UserID), portrait.DimensionEQ(portrait.DimensionBasic)).
		Only(ctx)
	if err != nil {
		return "", nil, err
	}

	// 查询是否是第一次创建
	stepFirst, err := r.data.ent_client.Step.Query().
		Where(step.RefTargetIDEQ(t.ID), step.TypeNEQ(step.TypeDir)).
		Order(ent.Asc(step.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		return "", nil, err
	}
	if stepFirst.ID == payload.StepID {
		// 第一次创建，根据目标设定时间和第一次打卡时间算出需要增加的果断值（0-10）。
		targetCreateTime := s.Edges.Target.CreatedAt
		stepFirstTime := stepFirst.CreatedAt
		duration := stepFirstTime - targetCreateTime
		// 一天内，增加10分， 两天内，增加5分， 三天内，增加2分， 超过3天，增加0分
		durationDays := decimal.NewFromInt(duration).Div(decimal.NewFromInt(24 * 3600))
		if durationDays.LessThanOrEqual(decimal.NewFromInt(1)) {
			portraitBasic.Value[objects.PortraitBasicDecisiveness] = cast.ToInt(portraitBasic.Value[objects.PortraitBasicDecisiveness]) + 10
		} else if durationDays.LessThanOrEqual(decimal.NewFromInt(2)) {
			portraitBasic.Value[objects.PortraitBasicDecisiveness] = cast.ToInt(portraitBasic.Value[objects.PortraitBasicDecisiveness]) + 5
		} else if durationDays.LessThanOrEqual(decimal.NewFromInt(3)) {
			portraitBasic.Value[objects.PortraitBasicDecisiveness] = cast.ToInt(portraitBasic.Value[objects.PortraitBasicDecisiveness]) + 2
		}
	}

	if s.IsChallenge {
		// 挑战性的，增加毅力值1。
		portraitBasic.Value[objects.PortraitBasicPerseverance] = cast.ToInt(portraitBasic.Value[objects.PortraitBasicPerseverance]) + 1
	} else {
		// 非挑战性的，增加耐心值1。
		portraitBasic.Value[objects.PortraitBasicPatience] = cast.ToInt(portraitBasic.Value[objects.PortraitBasicPatience]) + 1
	}

	_, err = r.data.ent_client.Portrait.UpdateOneID(portraitBasic.ID).
		SetValue(portraitBasic.Value).
		Save(ctx)
	if err != nil {
		return "", nil, err
	}

	// unix时间戳转时间
	stepTime := time.Unix(s.CreatedAt, 0).Local()
	// 需要同步在step_rates表中新建打卡记录，weighted_value设置为默认值1
	_, err = r.data.ent_client.StepRate.Create().
		SetUserID(t.UserID).
		SetTopTargetID(topTarget.ID).
		SetTargetID(s.Edges.Target.ID).
		SetStepID(s.ID).
		SetWeightedValue(0).
		SetTargetReasonableness(0).
		SetTargetClarity(0).
		SetTargetAchievement(0).
		SetReflectionImprovement(0).
		SetInnovation(0).
		SetBasicReliability(0).
		SetSkillImprovement(0).
		SetDifficulty(0).
		SetDate(stepTime).
		Save(ctx)
	if err != nil {
		return "", nil, err
	}

	// 算出打卡频率和持续性指标
	// TODO:

	return t.UserID,
	[]*objects.PortraitchangeType{
		{Type: "portrait", Scope: []string{portrait.DimensionBasic.String()}},
	}, nil
}

// 负面评价：评价在0-10之间，但是允许存在负面评价，所以需要统一-5，即5分为中间值，低于5分为负面评价。
// ● 每次积累评价之后，需要算出weighted_value并更新；如果存在的话，同时更新dimension_value
// ● 根据评价的目标合理性、目标明确性、目标达成度，增加到个人素质表中
// ● 每次自己认为是有难度则增加挑战心态值(可能是负面评价)
// ● 反思与改进、创新性、基础牢靠、技能提升增肌到个人素质表
func (r statisticsRepo) HandleStepComment(ctx context.Context, task *asynq.Task) (uid string, portraitChangeTypes []*objects.PortraitchangeType, err error) {
	var payload objects.StepCommentPayload
	err = json.Unmarshal(task.Payload(), &payload)
	if err != nil {
		return "", nil, err
	}

	t, err := r.stepRepo.GetTargetByStepIDRecursively(ctx, payload.StepID, 0)
	if err != nil {
		return "", nil, err
	}

	topTarget, err := r.stepRepo.GetTopTargetByTargetID(ctx, t.ID)
	if err != nil {
		return "", nil, err
	}

	err = r.CheckStatistics(ctx, t.UserID)
	if err != nil {
		return "", nil, err
	}

	s, err := r.data.ent_client.Step.Query().
		Where(step.ID(payload.StepID)).
		Only(ctx)
	if err != nil {
		return "", nil, err
	}

	// 获取评论
	var comment *objects.Comment
	var commentMap map[string]any
	if payload.CommentType == "teacher" {
		commentMap = s.TeacherComment
	} else if payload.CommentType == "parent" {
		commentMap = s.ParentComment
	} else if payload.CommentType == "friend" {
		commentMap = s.FriendComment
	}
	comment, err = objects.NewComment(commentMap)
	if err != nil {
		return "", nil, err
	}

	if comment == nil {
		return "", nil, fmt.Errorf("comment is nil")
	}

	weightedValue := comment.Data.WeightedValue - 5
	targetReasonableness := comment.Data.Target.TargetReasonableness - 5
	targetClarity := comment.Data.Target.TargetClarity - 5
	targetAchievement := comment.Data.Target.TargetAchievement - 5
	reflectionImprovement := comment.Data.Quality.ReflectionImprovement - 5
	innovation := comment.Data.Quality.Innovation - 5
	basicReliability := comment.Data.Quality.BasicReliability - 5
	skillImprovement := comment.Data.Quality.SkillImprovement - 5
	difficulty := comment.Data.Quality.Difficulty - 5

	// 更新step_rates表
	_, err = r.data.ent_client.StepRate.Update().
		Where(steprate.StepID(payload.StepID)).
		AddWeightedValue(weightedValue).
		AddTargetReasonableness(targetReasonableness).
		AddTargetClarity(targetClarity).
		AddTargetAchievement(targetAchievement).
		AddReflectionImprovement(reflectionImprovement).
		AddInnovation(innovation).
		AddBasicReliability(basicReliability).
		AddSkillImprovement(skillImprovement).
		AddDifficulty(difficulty).
		Save(ctx)
	if err != nil {
		return "", nil, err
	}

	// 更新portrait表，自律性
	if s.IsChallenge {
		// 挑战心态（迎难而上，不惧困难）
		portraitSelfDiscipline, err := r.data.ent_client.Portrait.Query().
			Where(portrait.UserID(t.UserID), portrait.DimensionEQ(portrait.DimensionSelfDiscipline)).
			Only(ctx)
		if err != nil {
			return "", nil, err
		}
		portraitSelfDiscipline.Value[objects.PortraitSelfDisciplineChallengeAttitude] =
			cast.ToInt(portraitSelfDiscipline.Value[objects.PortraitSelfDisciplineChallengeAttitude]) + int(difficulty)
		_, err = r.data.ent_client.Portrait.UpdateOneID(portraitSelfDiscipline.ID).
			SetValue(portraitSelfDiscipline.Value).
			Save(ctx)
		if err != nil {
			return "", nil, err
		}
	}

	// 更新portrait表，目标设定与执行能力
	portraitTargetAndExecution, err := r.data.ent_client.Portrait.Query().
		Where(portrait.UserID(t.UserID), portrait.DimensionEQ(portrait.DimensionTargetAndExecution)).
		Only(ctx)
	if err != nil {
		return "", nil, err
	}
	portraitTargetAndExecution.Value[objects.PortraitTargetAndExecutionGoalReasonableness] =
		cast.ToInt(portraitTargetAndExecution.Value[objects.PortraitTargetAndExecutionGoalReasonableness]) + int(targetReasonableness)
	portraitTargetAndExecution.Value[objects.PortraitTargetAndExecutionGoalClarity] =
		cast.ToInt(portraitTargetAndExecution.Value[objects.PortraitTargetAndExecutionGoalClarity]) + int(targetClarity)
	portraitTargetAndExecution.Value[objects.PortraitTargetAndExecutionGoalAchievement] =
		cast.ToInt(portraitTargetAndExecution.Value[objects.PortraitTargetAndExecutionGoalAchievement]) + int(targetAchievement)
	_, err = r.data.ent_client.Portrait.UpdateOneID(portraitTargetAndExecution.ID).
		SetValue(portraitTargetAndExecution.Value).
		Save(ctx)
	if err != nil {
		return "", nil, err
	}

	// 更新portrait表，学习与成长能力
	portraitLearningAndGrowth, err := r.data.ent_client.Portrait.Query().
		Where(portrait.UserID(t.UserID), portrait.DimensionEQ(portrait.DimensionLearningAndGrowth)).
		Only(ctx)
	if err != nil {
		return "", nil, err
	}
	portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthReflectionAndImprovement] =
		cast.ToInt(portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthReflectionAndImprovement]) + int(reflectionImprovement)
	portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthInnovativeMethod] =
		cast.ToInt(portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthInnovativeMethod]) + int(innovation)
	portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthBasicSolid] =
		cast.ToInt(portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthBasicSolid]) + int(basicReliability)
	portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthSkillImprovement] =
		cast.ToInt(portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthSkillImprovement]) + int(skillImprovement)
	if s.IsChallenge {
		portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthChallengeAttitude] =
			cast.ToInt(portraitLearningAndGrowth.Value[objects.PortraitLearningAndGrowthChallengeAttitude]) + int(difficulty)
	}
	_, err = r.data.ent_client.Portrait.UpdateOneID(portraitLearningAndGrowth.ID).
		SetValue(portraitLearningAndGrowth.Value).
		Save(ctx)
	if err != nil {
		return "", nil, err
	}

	return t.UserID, 
	[]*objects.PortraitchangeType{
		{Type: "portrait", Scope: []string{
			portrait.DimensionSelfDiscipline.String(),
			portrait.DimensionTargetAndExecution.String(),
			portrait.DimensionLearningAndGrowth.String(),
		}},
		{Type: "target", Scope: []string{cast.ToString(topTarget.ID)}},
	}, nil
}
