package biz

import (
	"context"

	"github.com/daymenu/gostudy/examples/ent/ent"
	"github.com/daymenu/gostudy/examples/ent/ent/card"
)

// CreateCard create card
func CreateCard(ctx context.Context, client *ent.Client) (*ent.Card, error) {
	return client.Card.
		Create().
		SetAmount(7.8).
		SetCardID("dd").
		Save(ctx)
}

// QueryCard query a card
func QueryCard(ctx context.Context, client *ent.Client) (*ent.Card, error) {
	return client.Card.Query().Where(card.CardIDEQ("dd")).First(ctx)
}

type CardAmount struct {
	CardID string  `json:"card_id"`
	Amount float64 `json:"sum"`
	Count  int     `json:"count"`
}

// QueryCardAmount query a card
func QueryCardAmount(ctx context.Context, client *ent.Client) (*[]CardAmount, error) {
	var v []CardAmount
	err := client.Card.Query().
		Where(card.CardIDEQ("dd")).
		GroupBy(card.FieldCardID).
		Aggregate(ent.Sum(card.FieldAmount), ent.Count()).
		Scan(ctx, &v)
	return &v, err
}
