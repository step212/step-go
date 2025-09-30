package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Portrait holds the schema definition for the Portrait entity.
type Portrait struct {
	ent.Schema
}

// Fields of the Portrait.
func (Portrait) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").Unique().Comment("自增ID"),
		field.String("user_id").Comment("User ID"),
		field.Enum("dimension").Values("basic", "self_discipline", "target_and_execution", "learning_and_growth").Comment("维度"),
		field.JSON("value", map[string]any{}).Comment("值"),
	}
}

// Edges of the Portrait.
func (Portrait) Edges() []ent.Edge {
	return nil
}
