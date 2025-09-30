package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Step holds the schema definition for the Step entity.
type Step struct {
	ent.Schema
}

// Fields of the Step.
func (Step) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").Unique().Comment("自增ID"),
		field.String("title").MaxLen(50).Optional().Comment("名称"),
		field.String("description").MaxLen(500).Optional().Comment("描述"),
		field.Bool("is_challenge").Default(false).Comment("是否挑战"),
		field.JSON("teacher_comment", map[string]any{}).Optional().Comment("老师评论"),
		field.JSON("parent_comment", map[string]any{}).Optional().Comment("家长评论"),
		field.JSON("friend_comment", map[string]any{}).Optional().Comment("朋友评论"),
		field.Enum("type").Values("image", "video", "audio", "dir").Comment("类型"),
		field.String("object_name").Optional().Unique().Comment("对象名"),
		field.Int64("created_at").Immutable().Default(time.Now().Local().Unix()).Comment("创建时间"),
		field.Uint64("ref_target_id").Optional().Comment("目标ID"),
		field.Uint64("parent_id").Optional().Comment("父ID"),
	}
}

// Edges of the Step.
func (Step) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("target", Target.Type).Ref("steps").Field("ref_target_id").Unique(),
		edge.To("children", Step.Type).From("parent").Field("parent_id").Unique(),
	}
}
