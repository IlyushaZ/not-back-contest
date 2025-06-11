package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/model"
)

type CheckoutRepository interface {
	Add(context.Context, ...model.Checkout) error
}

type CheckoutDatabase struct {
	DB *sql.DB
}

func (cd *CheckoutDatabase) Add(ctx context.Context, cos ...model.Checkout) error {
	if len(cos) == 0 {
		return nil
	}

	q := buildBatchQuery(len(cos))

	args := make([]any, 0, len(cos)*5)
	for _, co := range cos {
		code := sql.NullString{String: co.Code, Valid: co.Code != ""}
		errMsg := sql.NullString{String: co.Error, Valid: co.Error != ""}

		args = append(args, co.UserID, co.ItemID, co.CreatedAt, code, errMsg)
	}

	res, err := cd.DB.ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("can't insert checkouts: %w", err)
	}

	if affected, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("can't get affected rows: %w", err)
	} else if int(affected) != len(cos) {
		return fmt.Errorf("expected %d records to be inserted, got %d", len(cos), affected)
	}

	return nil
}

func buildBatchQuery(rows int) string {
	sb := strings.Builder{}
	sb.WriteString("insert into checkouts (user_id, item_id, created_at, code, error) values ")

	phs := make([]string, 0, rows)

	for i := range rows {
		phs = append(phs, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", i*5+1, i*5+2, i*5+3, i*5+4, i*5+5))
	}

	sb.WriteString(strings.Join(phs, ","))
	return sb.String()
}

type CheckoutBatchingDatabase struct {
	db        *sql.DB
	buffer    []model.Checkout
	ticker    *time.Ticker
	batchSize int
	mu        sync.Mutex

	*CheckoutDatabase
}

func NewCheckoutBatchingDatabase(db *sql.DB, batchSize int, flushInterval time.Duration) *CheckoutBatchingDatabase {
	return &CheckoutBatchingDatabase{
		db:        db,
		buffer:    make([]model.Checkout, 0, batchSize),
		ticker:    time.NewTicker(flushInterval),
		batchSize: batchSize,

		CheckoutDatabase: &CheckoutDatabase{db},
	}
}

func (cd *CheckoutBatchingDatabase) Add(ctx context.Context, cos ...model.Checkout) error {
	if len(cos) == 0 {
		return nil
	}

	cd.mu.Lock()
	cd.buffer = append(cd.buffer, cos...)
	shouldFlush := len(cd.buffer) >= cd.batchSize
	cd.mu.Unlock()

	if shouldFlush {
		go func() {
			if err := cd.flush(); err != nil {
				slog.Error("can't flush buffer", slog.Any("error", err))
			}
		}()
	}

	return nil
}

func (cd *CheckoutBatchingDatabase) flush() error {
	cd.mu.Lock()
	if len(cd.buffer) == 0 {
		cd.mu.Unlock()
		return nil
	}

	batch := make([]model.Checkout, len(cd.buffer))
	copy(batch, cd.buffer)
	cd.buffer = cd.buffer[:0]
	cd.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()

	// TODO: add retries?
	if err := cd.CheckoutDatabase.Add(ctx, batch...); err != nil {
		return fmt.Errorf("can't insert batch: %w", err)
	}

	return nil
}
