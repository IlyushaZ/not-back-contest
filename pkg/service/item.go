package service

import (
	"context"
	"fmt"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/database"
	"github.com/IlyushaZ/not-back-contest/pkg/model"
)

type Item interface {
	Checkout(ctx context.Context, userID, itemID int) (string, error)
	Purchase(ctx context.Context, code model.CheckoutCode) error
	ListPage(ctx context.Context, pageNum, pageSize int) ([]model.Item, int, error)
}

// ItemGeneric represents an implementation of Item interface containing core logics
// which can be wrapped in other implementations contained in item_*.go.
type ItemGeneric struct {
	ItemRepository  database.ItemRepository
	CheckoutTimeout time.Duration
}

func (ig *ItemGeneric) Checkout(ctx context.Context, userID, itemID int) (string, error) {
	// unfortunately, the logic has to be placed in repo, because it relies on database transaction
	code, err := ig.ItemRepository.Checkout(ctx, userID, itemID, ig.CheckoutTimeout)
	if err != nil {
		return "", fmt.Errorf("can't checkout item in DB: %w", err)
	}

	return code, nil
}

func (ig *ItemGeneric) Purchase(ctx context.Context, code model.CheckoutCode) error {
	return ig.ItemRepository.Purchase(ctx, code)
}

func (ig *ItemGeneric) ListPage(ctx context.Context, pageNum, pageSize int) ([]model.Item, int, error) {
	return ig.ItemRepository.GetPage(ctx, pageNum, pageSize)
}
