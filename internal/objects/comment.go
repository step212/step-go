package objects

import "encoding/json"

// comment v1.0
/* {
	"version": "v1.0",
	"data":{
	  "weighted_value": 5,
	  "target": {
		"target_reasonableness": 8,
		"target_clarity": 8,
		"target_achievement": 8
	  },
	  "quality":{
		"reflection_improvement": 5,
		"innovation": 5,
		"basic_reliability": 5,
		"skill_improvement": 5,
		"difficulty": 5
	  }
	}
   } */

type Comment struct {
	Version string `json:"version"`
	Data    struct {
		WeightedValue float64 `json:"weighted_value"`
		Target        struct {
			TargetReasonableness int32 `json:"target_reasonableness"`
			TargetClarity        int32 `json:"target_clarity"`
			TargetAchievement    int32 `json:"target_achievement"`
		} `json:"target"`
		Quality struct {
			ReflectionImprovement int32 `json:"reflection_improvement"`
			Innovation            int32 `json:"innovation"`
			BasicReliability      int32 `json:"basic_reliability"`
			SkillImprovement      int32 `json:"skill_improvement"`
			Difficulty            int32 `json:"difficulty"`
		} `json:"quality"`
	} `json:"data"`
}

func NewComment(commentMap map[string]any) (*Comment, error) {
	jsonComment, err := json.Marshal(commentMap)
	if err != nil {
		return nil, err
	}

	comment := &Comment{}
	err = json.Unmarshal(jsonComment, comment)
	if err != nil {
		return nil, err
	}
	return comment, nil
}
