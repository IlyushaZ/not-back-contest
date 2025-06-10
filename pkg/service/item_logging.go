package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/model"
)

type ItemLogging struct {
	Item
}

func (il *ItemLogging) Checkout(ctx context.Context, userID, itemID int) (code string, err error) {
	defer func(t0 time.Time) {
		log := slog.With(
			slog.Int("user_id", userID),
			slog.Int("item_id", itemID),
			slog.String("resp", "HIDDEN"),
			slog.String("delay", time.Since(t0).String()),
		)

		if err != nil {
			log.Error("failed to checkout item", slog.Any("error", err))
		} else {
			log.Debug("item checked out")
		}
	}(time.Now())

	return il.Item.Checkout(ctx, userID, itemID)
}

func (il *ItemLogging) Purchase(ctx context.Context, code model.CheckoutCode) (err error) {
	defer func(t0 time.Time) {
		log := slog.With(
			slog.String("code", code.String()),
			slog.String("delay", time.Since(t0).String()),
		)

		if err != nil {
			log.Error("failed to purchase item", slog.Any("error", err))
		} else {
			log.Debug("item purchased")
		}
	}(time.Now())

	return il.Item.Purchase(ctx, code)
}
