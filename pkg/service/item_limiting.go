package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/IlyushaZ/not-back-contest/pkg/limiter"
	"github.com/IlyushaZ/not-back-contest/pkg/model"
)

var ErrLimitExceeded = errors.New("used exceeded his limit")

// ItemLimiting is a wrapper over Item service
// which makes sure that user can make no more than LimitPerUser checkout requests per sale.
//
// If failed to check limits, the behavior depends on FailOpen flag. If set, current request is allowed.
// Otherwise, an error will be returned.
type ItemLimiting struct {
	Item

	Limiter  *limiter.Limiter
	FailOpen bool
}

func (ic *ItemLimiting) Checkout(ctx context.Context, userID, itemID int) (code string, err error) {
	exceeded, err := ic.Limiter.LimitExceeded(ctx, userID)
	if err != nil {
		if !ic.FailOpen {
			return "", fmt.Errorf("can't check if limit exceeded: %w", err)
		}

		slog.Error("can't check if limit exceeded", slog.Any("error", err))
	}

	if exceeded {
		return "", ErrLimitExceeded
	}

	return ic.Item.Checkout(ctx, userID, itemID)
}

func (ic *ItemLimiting) Purchase(ctx context.Context, code model.CheckoutCode) (err error) {
	err = ic.Item.Purchase(ctx, code)
	if err != nil {
		return
	}

	if _, err := ic.Limiter.Increment(ctx, code.UserID); err != nil {
		slog.Error("can't increment user's limit", slog.Any("error", err))
	}

	return err
}
