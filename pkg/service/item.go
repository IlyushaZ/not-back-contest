package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	ItemRepository     database.ItemRepository
	CheckoutRepository database.CheckoutRepository
	CheckoutTimeout    time.Duration
}

func (ig *ItemGeneric) Checkout(ctx context.Context, userID, itemID int) (code string, err error) {
	cc := model.CheckoutCode{UserID: userID, ItemID: itemID}
	cc.GenerateRand()
	code = cc.String()

	defer func() {
		if !shouldSaveCheckout(err) {
			return
		}

		co := model.Checkout{
			Base:   model.Base{CreatedAt: time.Now()},
			UserID: userID,
			ItemID: itemID,
		}

		if err == nil {
			co.Code = code
		} else {
			co.Error = err.Error()
		}

		if err := ig.CheckoutRepository.Add(ctx, co); err != nil {
			slog.Error("can't save checkout to DB", slog.Any("error", err))
		}
	}()

	err = ig.ItemRepository.Checkout(ctx, userID, itemID, cc, ig.CheckoutTimeout)
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

func shouldSaveCheckout(err error) bool {
	return err == nil || errors.Is(err, model.ErrItemUnavailable)
}
