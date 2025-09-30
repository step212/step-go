package biz

import (
	"context"
	"errors"
	"fmt"
	"step/internal/data/ent"
	"step/internal/data/ent/portrait"
	"step/internal/data/ent/steprate"
	"step/internal/utils"
	"time"

	stepApi "step/api/step/v1"

	"entgo.io/ent/dialect/sql"
	"github.com/Jeffail/gabs/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
)

type PortraitUsecase struct {
	stepRepo         StepRepo
	entClient        *ent.Client
	log              *log.Helper
}

func NewPortraitUsecase(
	stepRepo StepRepo,
	entClient *ent.Client,
	logger log.Logger,
) *PortraitUsecase {
	return &PortraitUsecase{
		stepRepo:         stepRepo,
		entClient:        entClient,
		log:              log.NewHelper(logger, log.WithMessageKey("stepUsecase")),
	}
}

func (uc *PortraitUsecase) GetPortraitBasic(ctx context.Context, req *stepApi.GetPortraitBasicRequest) (*stepApi.GetPortraitBasicReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	query := uc.entClient.Portrait.Query().Where(portrait.UserIDEQ(uid))
	if req.Dimension != "" {
		query = query.Where(portrait.DimensionEQ(portrait.Dimension(req.Dimension)))
	}

	portraits, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	if len(portraits) == 0 {
		return nil, errors.New("portrait not found")
	}

	value := gabs.New()
	for _, portrait := range portraits {
		value.Set(portrait.Value, portrait.Dimension.String())
	}

	return &stepApi.GetPortraitBasicReply{
		Dimension: req.Dimension,
		Value:     value.String(),
	}, nil
}

func (uc *PortraitUsecase) GetPortraitStepRate(ctx context.Context, req *stepApi.GetPortraitStepRateRequest) (*stepApi.GetPortraitStepRateReply, error) {
	uid := utils.GetUid(ctx)
	if uid == "" {
		return nil, errors.New("uid is empty")
	}

	if req.TopTargetId == 0 {
		return nil, errors.New("top_target_id is empty")
	}

	if req.StatUnit == "" {
		return nil, errors.New("stat_unit is empty")
	}

	query := uc.entClient.StepRate.Query().Where(steprate.UserIDEQ(uid), steprate.TopTargetIDEQ(req.TopTargetId))
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02 15:04:05", req.StartDate + " 00:00:00")
		if err != nil {
			return nil, err
		}
		query = query.Where(steprate.DateGTE(startDate))
	}
	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02 15:04:05", req.EndDate + " 23:59:59")
		if err != nil {
			return nil, err
		}
		query = query.Where(steprate.DateLTE(endDate))
	}

	// aggregation by unit
	var aggData []struct {
		Unit            string 	    		`json:"unit"`
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

	var unitExpr string
	if req.StatUnit == "day" {
		unitExpr = fmt.Sprintf("DATE_FORMAT(%s, '%s')", steprate.FieldDate, "%Y-%m-%d")
	} else if req.StatUnit == "week" {
		unitExpr = fmt.Sprintf("DATE_FORMAT(%s, '%s')", steprate.FieldDate, "%x-W%v")
	} else if req.StatUnit == "month" {
		unitExpr = fmt.Sprintf("DATE_FORMAT(%s, '%s')", steprate.FieldDate, "%Y-%m")
	}
	err := query.Modify(func(s *sql.Selector) {
		s.Select(
			sql.As(unitExpr, "unit"),
			sql.As(sql.Avg(s.C(steprate.FieldWeightedValue)), "weight_value_avg"),
			sql.As(sql.Avg(s.C(steprate.FieldTargetReasonableness)), "target_reasonableness_avg"),
			sql.As(sql.Avg(s.C(steprate.FieldTargetClarity)), "target_clarity_avg"),
			sql.As(sql.Avg(s.C(steprate.FieldTargetAchievement)), "target_achievement_avg"),
			sql.As(sql.Avg(s.C(steprate.FieldReflectionImprovement)), "reflection_improvement_avg"),
			sql.As(sql.Avg(s.C(steprate.FieldInnovation)), "innovation_avg"),
			sql.As(sql.Avg(s.C(steprate.FieldBasicReliability)), "basic_reliability_avg"),
			sql.As(sql.Avg(s.C(steprate.FieldSkillImprovement)), "skill_improvement_avg"),
			sql.As(sql.Avg(s.C(steprate.FieldDifficulty)), "difficulty_avg"),
		).
		GroupBy("unit").
		OrderBy(sql.Asc("unit"))
	}).Scan(ctx, &aggData)
	if err != nil {
		return nil, err
	}

	value := gabs.New()
	value.Array("weighted_value")
	value.Array("target_reasonableness")
	value.Array("target_clarity")
	value.Array("target_achievement")
	value.Array("reflection_improvement")
	value.Array("innovation")
	value.Array("basic_reliability")
	value.Array("skill_improvement")
	value.Array("difficulty")
	for _, row := range aggData {
		weightedValue := gabs.New()
		weightedValue.Set(decimal.NewFromFloat(row.WeightValueAvg).Round(2).InexactFloat64(), row.Unit)
		value.ArrayAppend(weightedValue, "weighted_value")
		targetReasonableness := gabs.New()
		targetReasonableness.Set(decimal.NewFromFloat(row.TargetReasonablenessAvg).Round(2).InexactFloat64(), row.Unit)
		value.ArrayAppend(targetReasonableness, "target_reasonableness")
		targetClarity := gabs.New()
		targetClarity.Set(decimal.NewFromFloat(row.TargetClarityAvg).Round(2).InexactFloat64(), row.Unit)
		value.ArrayAppend(targetClarity, "target_clarity")
		targetAchievement := gabs.New()
		targetAchievement.Set(decimal.NewFromFloat(row.TargetAchievementAvg).Round(2).InexactFloat64(), row.Unit)
		value.ArrayAppend(targetAchievement, "target_achievement")
		reflectionImprovement := gabs.New()
		reflectionImprovement.Set(decimal.NewFromFloat(row.ReflectionImprovementAvg).Round(2).InexactFloat64(), row.Unit)
		value.ArrayAppend(reflectionImprovement, "reflection_improvement")
		innovation := gabs.New()
		innovation.Set(decimal.NewFromFloat(row.InnovationAvg).Round(2).InexactFloat64(), row.Unit)
		value.ArrayAppend(innovation, "innovation")
		basicReliability := gabs.New()
		basicReliability.Set(decimal.NewFromFloat(row.BasicReliabilityAvg).Round(2).InexactFloat64(), row.Unit)
		value.ArrayAppend(basicReliability, "basic_reliability")
		skillImprovement := gabs.New()
		skillImprovement.Set(decimal.NewFromFloat(row.SkillImprovementAvg).Round(2).InexactFloat64(), row.Unit)
		value.ArrayAppend(skillImprovement, "skill_improvement")
		difficulty := gabs.New()
		difficulty.Set(decimal.NewFromFloat(row.DifficultyAvg).Round(2).InexactFloat64(), row.Unit)
		value.ArrayAppend(difficulty, "difficulty")
	}

	return &stepApi.GetPortraitStepRateReply{
		TopTargetId:   req.TopTargetId,
		StatUnit:      req.StatUnit,
		Value:         value.String(),
	}, nil
}



