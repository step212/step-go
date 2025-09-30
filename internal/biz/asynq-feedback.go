package biz

import (
	"context"
	"encoding/json"
	"step/internal/data/ent"
	"step/internal/data/ent/award"
	"step/internal/data/ent/portrait"
	"step/internal/data/ent/steprate"
	"step/internal/objects"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
	"github.com/spf13/cast"
)

// AsynqFeedbackUsecase is a AsynqFeedback usecase.
type AsynqFeedbackUsecase struct {
	log            *log.Helper
	entClient      *ent.Client
}

// NewAsynqFeedbackUsecase new a AsynqFeedback usecase.
func NewAsynqFeedbackUsecase(logger log.Logger, entClient *ent.Client) *AsynqFeedbackUsecase {
	return &AsynqFeedbackUsecase{
		log:            log.NewHelper(logger, log.WithMessageKey("asynqFeedbackUsecase")),
		entClient:      entClient,
	}
}

func (uc *AsynqFeedbackUsecase) handleFeedbackPortraitChange(ctx context.Context, uid string, change *objects.PortraitchangeType) error {
	for _, scope := range change.Scope {
		curPortrait, err := uc.entClient.Portrait.Query().
			Where(portrait.UserID(uid)).
			Where(portrait.DimensionEQ(portrait.Dimension(scope))).
			First(ctx)
		if err != nil {
			uc.log.Errorf("handleFeedbackPortraitChange: %v", err)
			continue
		}
		curDimensionValue := curPortrait.Value
		
		awds, err := uc.entClient.Award.Query().
			Where(award.UserID(uid)).
			Where(award.StatusEQ(award.StatusSetted)).
			Where(award.TargetTypeEQ(award.TargetTypePortrait)).
			Where(award.ScopeEQ(scope)).
			All(ctx)
		if err != nil {
			uc.log.Errorf("handleFeedbackPortraitChange: %v", err)
			continue
		}

		for _, awd := range awds {
			curValue := cast.ToInt32(curDimensionValue[awd.Dimension])
			if curValue >= awd.Threshold {
				_, err := uc.entClient.Award.UpdateOne(awd).
					SetStatus(award.StatusAchieved).
					SetAchievedAt(time.Now().Unix()).
					Save(ctx)
				if err != nil {
					uc.log.Errorf("handleFeedbackPortraitChange: %v", err)
					continue
				}
			}
		}
	}

	return nil
}

func (uc *AsynqFeedbackUsecase) handleFeedbackTargetChange(ctx context.Context, uid string, change *objects.PortraitchangeType) error {
	for _, scope := range change.Scope {
		topTargetId := cast.ToUint64(scope)

		var aggData []struct {
			WeightValueAvg  float64  			`json:"weight_value_avg"`
			TargetReasonablenessAvg  float64  	`json:"target_reasonableness_avg"`
			TargetClarityAvg  float64  			`json:"target_clarity_avg"`
			TargetAchievementAvg  float64  		`json:"target_achievement_avg"`
			ReflectionImprovementAvg  float64  	`json:"reflection_improvement_avg"`
			InnovationAvg  float64  			`json:"innovation_avg"`
			BasicReliabilityAvg  float64  		`json:"basic_reliability_avg"`
			SkillImprovementAvg  float64  		`json:"skill_improvement_avg"`
			DifficultyAvg  float64  			`json:"difficulty_avg"`
		}
		err := uc.entClient.StepRate.Query().Where(steprate.TopTargetIDEQ(topTargetId)).Modify(func(s *sql.Selector) {
			s.Select(
				sql.As(sql.Avg(s.C(steprate.FieldWeightedValue)), "weight_value_avg"),
				sql.As(sql.Avg(s.C(steprate.FieldTargetReasonableness)), "target_reasonableness_avg"),
				sql.As(sql.Avg(s.C(steprate.FieldTargetClarity)), "target_clarity_avg"),
				sql.As(sql.Avg(s.C(steprate.FieldTargetAchievement)), "target_achievement_avg"),
				sql.As(sql.Avg(s.C(steprate.FieldReflectionImprovement)), "reflection_improvement_avg"),
				sql.As(sql.Avg(s.C(steprate.FieldInnovation)), "innovation_avg"),
				sql.As(sql.Avg(s.C(steprate.FieldBasicReliability)), "basic_reliability_avg"),
				sql.As(sql.Avg(s.C(steprate.FieldSkillImprovement)), "skill_improvement_avg"),
				sql.As(sql.Avg(s.C(steprate.FieldDifficulty)), "difficulty_avg"),
			)
		}).Scan(ctx, &aggData)
		if err != nil {
			uc.log.Errorf("handleFeedbackTargetChange: %v", err)
			continue
		}

		curValue := make(map[string]float64)
		curValue["weighted_value"] = aggData[0].WeightValueAvg
		curValue["target_reasonableness"] = aggData[0].TargetReasonablenessAvg
		curValue["target_clarity"] = aggData[0].TargetClarityAvg
		curValue["target_achievement"] = aggData[0].TargetAchievementAvg
		curValue["reflection_improvement"] = aggData[0].ReflectionImprovementAvg
		curValue["innovation"] = aggData[0].InnovationAvg
		curValue["basic_reliability"] = aggData[0].BasicReliabilityAvg
		curValue["skill_improvement"] = aggData[0].SkillImprovementAvg
		curValue["difficulty"] = aggData[0].DifficultyAvg

		awds, err := uc.entClient.Award.Query().
			Where(award.UserID(uid)).
			Where(award.StatusEQ(award.StatusSetted)).
			Where(award.TargetTypeEQ(award.TargetTypeTarget)).
			Where(award.ScopeEQ(scope)).
			All(ctx)
		if err != nil {
			uc.log.Errorf("handleFeedbackTargetChange: %v", err)
			continue
		}

		for _, awd := range awds {
			curValue := curValue[awd.Dimension]
			if curValue >= float64(awd.Threshold) {
				_, err := uc.entClient.Award.UpdateOne(awd).
					SetStatus(award.StatusAchieved).
					SetAchievedAt(time.Now().Unix()).
					Save(ctx)
				if err != nil {
					uc.log.Errorf("handleFeedbackTargetChange: %v", err)
					continue
				}
			}
		}
	}

	return nil
}

func (uc *AsynqFeedbackUsecase) HandleFeedbackPortraitChange(ctx context.Context, task *asynq.Task) error {
	var payload objects.FeedbackPortraitChangePayload
	err := json.Unmarshal(task.Payload(), &payload)
	if err != nil {
		return err
	}

	uc.log.Infof("HandleFeedbackPortraitChange: %v", payload)

	for _, change := range payload.PortraitChangeTypes {
		if change.Type == "portrait" {
			uc.handleFeedbackPortraitChange(ctx, payload.UserID, change)
		} else if change.Type == "target" {
			uc.handleFeedbackTargetChange(ctx, payload.UserID, change)
		}
	}

	return nil
}
