package schema

import (
	"github.com/facebook/ent"
	"github.com/facebook/ent/schema/field"
	"github.com/facebook/ent/schema/index"
	"github.com/facebook/ent/schema/mixin"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"). // 设置字段名字
					Default("").
					Comment("名字"),

		field.String("email"). // 设置字段名字
					Default("").
					Comment("邮箱"),

		field.String("password").
			Sensitive().
			Comment("密码"),
	}
}

// Indexs of the User
func (User) Indexs() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
		index.Fields("email"). // 定义index索引
					Unique(), //定义唯一索引
	}
}

// Mixin 定义通用字段
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
		StatusMixin{},
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
