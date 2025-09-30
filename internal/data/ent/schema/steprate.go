package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// StepRate holds the schema definition for the StepRate entity.
type StepRate struct {
	ent.Schema
}

// Fields of the StepRate.
func (StepRate) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").Unique().Comment("自增ID"),
		field.String("user_id").Comment("User ID"),
		field.Uint64("top_target_id").Comment("Top Target ID"),
		field.Uint64("target_id").Comment("Target ID"),
		field.Uint64("step_id").Unique().Comment("Step ID"),
		field.Float("weighted_value").Comment("加权值"),
		// json聚合困难，暂时不使用
		//field.JSON("dimension_value", map[string]any{}).Optional().Comment("维度值"),
		field.Int32("target_reasonableness").Comment("目标合理性"),
		field.Int32("target_clarity").Comment("目标明确性"),
		field.Int32("target_achievement").Comment("目标达成度"),
		field.Int32("reflection_improvement").Comment("反思与改进"),
		field.Int32("innovation").Comment("创新性"),
		field.Int32("basic_reliability").Comment("基础牢靠"),
		field.Int32("skill_improvement").Comment("技能提升"),
		field.Int32("difficulty").Comment("困难度"),
		field.Time("date").SchemaType(map[string]string{
			"mysql":    "date",
			"postgres": "date",
			"sqlite3":  "date",
		}).Comment("日期"),
	}
}

// Edges of the StepRate.
func (StepRate) Edges() []ent.Edge {
	return nil
}
