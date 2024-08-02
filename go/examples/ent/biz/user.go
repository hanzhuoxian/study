package biz

import (
	"context"
	"fmt"
	"log"

	"github.com/daymenu/gostudy/examples/ent/ent"
)

// CreateUser create user
func CreateUser(ctx context.Context, client *ent.Client) (*ent.User, error) {
	u, err := client.User.
		Create().
		SetName("ent").
		SetEmail("ent@facebook.com").
		SetStatus(1).
		SetPassword("ent").
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed create user: %s", err)
	}
	// client.Pet.Create().Save(ctx)
	log.Println("user was created:", u)
	return u, nil
}
