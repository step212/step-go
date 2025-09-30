package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ShowReserve holds the schema definition for the ShowReserve entity.
type ShowReserve struct {
	ent.Schema
}

// Fields of the ShowReserve.
func (ShowReserve) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").Unique().Comment("自增ID"),
		field.String("user_id").Comment("User ID"),
		field.Enum("status").Values("recommend", "reserved", "completed").Default("recommend").Comment("状态"),
		field.Strings("memories").Optional().Comment("记忆"),
		field.Int64("created_at").Immutable().Default(time.Now().Local().Unix()).Comment("创建时间"),
		field.Uint64("ref_show_id").Optional().Comment("关联的Show ID"),
	}
}

// Edges of the ShowReserve.
func (ShowReserve) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("show", Show.Type).Ref("show_reserves").Field("ref_show_id").Unique(),
	}
}
