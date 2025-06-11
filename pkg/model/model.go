package model

import (
	"time"
)

type Base struct {
	ID        int       `json:"id"` // int/serial used for simplicity, in prod env uuid is more preferrable
	CreatedAt time.Time `json:"created_at"`
}
