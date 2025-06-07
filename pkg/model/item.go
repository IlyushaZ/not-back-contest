package model

import (
	"database/sql"
)

type Item struct {
	Base
	Name          string         `json:"name"`
	SaleID        int            `json:"sale_id"`
	Sold          bool           `json:"sold"`
	ReservedUntil sql.NullTime   `json:"-"`
	ReservedBy    sql.NullInt64  `json:"-"`
	Code          sql.NullString `json:"-"` // for faster access when purchasing
}
