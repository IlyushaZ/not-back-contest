package model

import (
	"errors"
	"time"
)

var (
	ErrItemUnavailable = errors.New("item is unavailable for checkout")
	ErrSaleExpired     = errors.New("sale has expired")
	ErrCheckoutExpired = errors.New("checkout has expired")
)

type Base struct {
	ID        int       `json:"id"` // int/serial used for simplicity, in prod env uuid is more preferrable
	CreatedAt time.Time `json:"created_at"`
}
