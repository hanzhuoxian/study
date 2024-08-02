package biz

import (
	"context"
	"fmt"
	"log"

	"github.com/daymenu/gostudy/examples/ent/ent"
)

// CreateAdmin create admin
func CreateAdmin(ctx context.Context, client *ent.Client) (*ent.Admin, error) {
	u, err := client.Admin.
		Create().
		SetAge(18).
		SetName("ent").
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed create user: %s", err)
	}
	// client.Pet.Create().Save(ctx)
	log.Println("user was created:", u)
	return u, nil
}
