package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Target holds the schema definition for the Target entity.
type Target struct {
	ent.Schema
}

// Fields of the Target.
func (Target) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").Unique().Comment("自增ID"),
		field.String("user_id").Comment("User ID"),
		field.String("title").MaxLen(50).Comment("名称"),
		field.String("description").MaxLen(500).Comment("描述"),
		field.String("type").Default("default").Comment("类型"),
		field.Int64("created_at").Immutable().Default(time.Now().Local().Unix()).Comment("创建时间"),
		field.Int64("start_at").Optional().Comment("开始时间"),
		field.Int64("challenge_at").Optional().Comment("挑战时间"),
		field.Int64("done_at").Optional().Comment("完成时间"),
		field.Uint32("layer").Default(0).Comment("层级"),
		field.Enum("status").Values("init", "step", "step_hard", "done").Default("init").Comment("状态"),
		field.Uint64("parent_id").Optional().Comment("父ID"),
	}
}

// Edges of the Target.
func (Target) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", Target.Type).From("parent").Field("parent_id").Unique(),
		edge.To("steps", Step.Type),
	}
}
