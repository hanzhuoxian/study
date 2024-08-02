package schema

import (
	"errors"
	"regexp"
	"strings"

	"github.com/facebook/ent"
	"github.com/facebook/ent/dialect/entsql"
	"github.com/facebook/ent/schema/field"
)

// Group holds the schema definition for the Group entity.
type Group struct {
	ent.Schema
}

// Fields of the Group.
func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Match(regexp.MustCompile("[a-zA-Z_]+$")).
			MinLen(5).
			Validate(func(s string) error {
				if strings.ToLower(s) == s {
					return errors.New("group name must begin with uppercase")
				}
				return nil
			}),
		field.String("nillable_name").
			Optional().Nillable(),
		field.String("optional_name").
			Optional().
			StructTag(`gqlgen:"gql_name"`),
		field.Time("creation_date").
			Annotations(entsql.Annotation{
				Table: "my_group",
			}),
	}
}

// Edges of the Group.
func (Group) Edges() []ent.Edge {
	return nil
}
