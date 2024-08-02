package schema

import (
	"github.com/facebook/ent"
	"github.com/facebook/ent/schema/field"
	"github.com/facebook/ent/schema/mixin"
)

// StatusMixin 自定义字段结构体
type StatusMixin struct {
	mixin.Schema
}

// Fields 添加字段
func (StatusMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Int("status").
			Default(0),
	}
}
