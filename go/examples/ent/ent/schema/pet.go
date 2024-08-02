package schema

import (
	"github.com/facebook/ent"
	"github.com/facebook/ent/schema/field"
	"github.com/facebook/ent/schema/index"
	"github.com/facebook/ent/schema/mixin"
)

// Pet holds the schema definition for the Pet entity.
type Pet struct {
	ent.Schema
}

// Mixin 定义通用字段
func (Pet) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
		StatusMixin{},
	}
}

// Indexs 定义
func (Pet) Indexs() []ent.Index {
	return []ent.Index{
		index.Fields("name").Edges("user"),
	}
}

// Fields of the Pet.
func (Pet) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"). // 设置字段名字
					Default("").
					Comment("名字"),
	}
}

// Edges of the Pet.
func (Pet) Edges() []ent.Edge {
	return nil
}
