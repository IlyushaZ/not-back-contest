package model

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrItemUnavailable = errors.New("item is unavailable for checkout")
)

type Item struct {
	Base
	Name          string         `json:"name"`
	SaleID        int            `json:"sale_id"`
	SaleStart     time.Time      `json:"-"`
	SaleEnd       time.Time      `json:"-"`
	Sold          bool           `json:"sold"`
	ReservedUntil sql.NullTime   `json:"-"`
	ReservedBy    sql.NullInt64  `json:"-"`
	Code          sql.NullString `json:"-"`
}
