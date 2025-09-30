package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Show holds the schema definition for the Show entity.
type Show struct {
	ent.Schema
}

// Fields of the Show.
func (Show) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").Unique().Comment("自增ID"),
		field.String("type").Comment("Type"),
		field.String("poster").Optional().Comment("Poster"),
		field.Text("content").Comment("Markdown Content"),
		field.Strings("media_files").Optional().Comment("Media Files"),
		field.Int64("created_at").Immutable().Default(time.Now().Local().Unix()).Comment("创建时间"),
	}
}

// Edges of the Show.
func (Show) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("show_reserves", ShowReserve.Type),
	}
}
