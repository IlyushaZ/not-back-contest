package model

import (
	"time"
)

const (
	SaleDuration = time.Hour
	ItemsPerSale = 10000
)

type Sale struct {
	Base
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}
