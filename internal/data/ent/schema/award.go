package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Award holds the schema definition for the Award entity.
type Award struct {
	ent.Schema
}

// Fields of the Award.
func (Award) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").Unique().Comment("自增ID"),
		field.String("user_id").Comment("User ID"),
		field.Enum("status").Values("setted", "achieved", "realized").Default("setted").Comment("Status"),
		field.String("description").Comment("Description"),
		field.Strings("setted_files").Optional().Comment("Setted Files"),
		field.Strings("realized_files").Optional().Comment("Realized Files"),
		field.Enum("target_type").Values("portrait", "target").Comment("设定的目标类型, 自身画像或者目标"),
		field.String("scope").Comment("Scope, top dimension for portrait type/top_target_id for target type"),
		field.String("dimension").Comment("维度"),
		field.Int32("threshold").Comment("阈值"),
		field.Int64("setted_at").Immutable().Default(time.Now().Local().Unix()).Comment("Setted At"),
		field.Int64("achieved_at").Optional().Comment("Achieved At"),
		field.Int64("realized_at").Optional().Comment("Realized At"),
	}
}

// Edges of the Award.
func (Award) Edges() []ent.Edge {
	return nil
}
