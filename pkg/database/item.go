package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/model"
)

type ItemRepository interface {
	// Checkout tries to reserve the item for given user for timeout seconds/minutes/hours.
	Checkout(ctx context.Context, userID, itemID int, code model.CheckoutCode, timeout time.Duration) error
	Purchase(ctx context.Context, code model.CheckoutCode) error
	GetPage(ctx context.Context, num, size int) ([]model.Item, int, error)
}

type ItemDatabase struct {
	db    *sql.DB
	stmts map[string]*sql.Stmt
}

func NewItemDatabase(db *sql.DB) (*ItemDatabase, error) {
	idb := &ItemDatabase{
		db,
		make(map[string]*sql.Stmt),
	}

	for _, s := range stmts {
		prepared, err := db.Prepare(s.query)
		if err != nil {
			return nil, fmt.Errorf("can't prepare query '%s': %w", s.name, err)
		}

		idb.stmts[s.name] = prepared
	}

	return idb, nil
}

type preparedStmt struct {
	name  string
	query string
}

var (
	stmts = []preparedStmt{
		{
			name: "checkout_item",
			query: `
				update items
				set reserved_by = $1, reserved_until = $2, code = $3
				where id = $4
				  and not sold
				  and sale_start < $5 and sale_end > $5
				  and (reserved_until is null or reserved_until < $5)
			`,
		},
		{
			name: "purchase_item",
			query: `
				update items
				set sold = true
				where id = $1
				  and not sold
				  and reserved_by = $2
				  and code = $3
				  and reserved_until > $4
				  and sale_start < $4
				  and sale_end > $4
			`,
		},
	}
)

func (i *ItemDatabase) Checkout(ctx context.Context, userID, itemID int, code model.CheckoutCode, checkoutTimeout time.Duration) error {
	now := time.Now()

	res, err := i.stmts["checkout_item"].ExecContext(ctx, userID, now.Add(checkoutTimeout), code.Rand, itemID, now)
	if err != nil {
		return fmt.Errorf("can't update item: %w", err)
	}

	if affected, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("can't get affected rows: %w", err)
	} else if affected != 1 {
		return model.ErrItemUnavailable
	}

	return nil
}

func (i *ItemDatabase) Purchase(ctx context.Context, code model.CheckoutCode) error {
	res, err := i.stmts["purchase_item"].ExecContext(ctx, code.ItemID, code.UserID, code.Rand, time.Now())
	if err != nil {
		return fmt.Errorf("can't update item's status: %w", err)
	}

	if affected, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("can't get affected rows: %w", err)
	} else if affected == 0 {
		return fmt.Errorf("either item or checkout does not exist: %w", ErrNotFound)
	}

	return nil
}

func (i *ItemDatabase) GetPage(ctx context.Context, num, size int) ([]model.Item, int, error) {
	q := `
		select count(*) from items
	`
	var total int
	if err := i.db.QueryRowContext(ctx, q).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("can't count items: %w", err)
	}

	offset := (num - 1) * size
	q = `
		select id, name, sale_id, sold, created_at
		from items
		order by id
		limit $1 offset $2
	`
	rows, err := i.db.QueryContext(ctx, q, size, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("can't query items: %w", err)
	}
	defer rows.Close()

	items := make([]model.Item, 0, size)
	for rows.Next() {
		var item model.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.SaleID, &item.Sold, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("can't scan item: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating over items: %w", err)
	}

	return items, total, nil
}

func shouldLogCheckout(err error) bool {
	return err == nil || errOneOf(err, model.ErrSaleExpired, model.ErrItemUnavailable)
}

func errOneOf(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
