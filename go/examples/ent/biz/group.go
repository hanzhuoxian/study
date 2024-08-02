package biz

import (
	"context"

	"github.com/daymenu/gostudy/examples/ent/ent"
	"github.com/daymenu/gostudy/examples/ent/ent/group"
)

// CreateGroup create group
func CreateGroup(ctx context.Context, client *ent.Client) (*ent.Group, error) {
	return client.Group.
		Create().
		SetName("Baaaa").
		SetNillableName("").
		Save(ctx)
}

// QueryGroup query group
func QueryGroup(ctx context.Context, client *ent.Client) (*ent.Group, error) {
	return client.Group.Query().Where(group.Name("Baaaa")).Only(ctx)
}
