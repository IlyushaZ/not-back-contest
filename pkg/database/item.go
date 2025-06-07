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
	Checkout(ctx context.Context, userID, itemID int, timeout time.Duration) (code string, err error)
	Purchase(ctx context.Context, code model.CheckoutCode) error
	GetPage(ctx context.Context, num, size int) ([]model.Item, int, error)
}

type ItemDatabase struct {
	DB *sql.DB
}

// checkoutItem represents a group of fields that needs to be selected during checkout process.
// it differs from model.Item because it also includes fields joined from other tables.
type checkoutItem struct {
	model.Item
	saleStart, saleEnd time.Time
}

func (i *ItemDatabase) Checkout(ctx context.Context, userID, itemID int, checkoutTimeout time.Duration) (code string, err error) {
	err = WithTx(i.DB, func(tx *sql.Tx) (err error) {
		now := time.Now()

		// when we're done, we should log the checkout attempt
		// TODO: may be move it outside the transaction or even make asynchronous?
		defer func() {
			if !shouldLogCheckout(err) {
				return
			}

			q := `
				insert into checkouts (user_id, item_id, created_at, code, error)
				values ($1, $2, $3, $4, $5)
			`

			e := sql.NullString{}
			if err != nil {
				e.String = err.Error()
				e.Valid = err.Error() != ""
			}

			c := sql.NullString{String: code, Valid: code != ""} // may be null if err occurred

			if _, coErr := tx.ExecContext(ctx, q, userID, itemID, now, c, e); coErr != nil {
				err = fmt.Errorf("can't log checkout attempt: %w. orig error: %w", coErr, err)
			}
		}()

		q := `
			select i.id, i.sold, i.reserved_until, i.reserved_by, i.code, s.start_at, s.end_at
			from items i
			join sales s on i.sale_id = s.id
			where i.id = $1
			for update
		`

		var ci checkoutItem

		err = tx.QueryRowContext(ctx, q, itemID).Scan(&ci.ID, &ci.Sold, &ci.ReservedUntil, &ci.ReservedBy, &ci.Code, &ci.saleStart, &ci.saleEnd)
		if err != nil {
			return fmt.Errorf("can't get item data: %w", mapError(err))
		}

		if ci.Sold {
			return model.ErrItemUnavailable
		}

		if now.Before(ci.saleStart) || now.After(ci.saleEnd) {
			return model.ErrSaleExpired
		}

		// item has fresh checkout
		if ci.ReservedUntil.Valid {
			ru := ci.ReservedUntil.Time

			if ru.After(now) && ru.Sub(now) < model.DefaultCheckoutTimeout {
				if rb := int(ci.ReservedBy.Int64); rb == userID { // it's us who had reserved the item before
					mc := model.CheckoutCode{rb, ci.ID, ci.Code.String}
					code = mc.String()
					return nil
				}

				return model.ErrItemUnavailable
			}
		}

		// generate code
		mc := model.CheckoutCode{UserID: userID, ItemID: itemID}
		mc.WithRand()
		code = mc.String() // code to return outside

		q = `
			update items
			set code = $1, reserved_until = $2, reserved_by = $3
			where id = $4
		`
		_, err = tx.ExecContext(
			ctx,
			q,
			mc.Rand,
			now.Add(checkoutTimeout),
			userID,
			itemID,
		)
		if err != nil {
			return fmt.Errorf("can't update item: %w", err)
		}

		return nil
	})

	return code, err
}

func (i *ItemDatabase) Purchase(ctx context.Context, code model.CheckoutCode) error {
	return WithTx(i.DB, func(tx *sql.Tx) error {
		const q = `
			with to_update as (
			    select i.id
			    from items i
			    join sales s on i.sale_id = s.id
			    where i.id = $1
			      and not i.sold
			      and i.reserved_by = $2
			      and i.code = $3
			      and i.reserved_until > $4
			      and s.start_at < $4
			      and s.end_at > $4
			    for update
			)
			update items
			set sold = true
			from to_update
			where items.id = to_update.id;
		`

		res, err := tx.ExecContext(ctx, q, code.ItemID, code.UserID, code.Rand, time.Now())
		if err != nil {
			return fmt.Errorf("can't update item's status: %w", err)
		}

		if affected, err := res.RowsAffected(); err != nil {
			return fmt.Errorf("can't get affected rows: %w", err)
		} else if affected == 0 {
			return fmt.Errorf("either item or checkout does not exist: %w", ErrNotFound)
		}

		return nil
	})
}

func (i *ItemDatabase) GetPage(ctx context.Context, num, size int) ([]model.Item, int, error) {
	q := `
		select count(*) from items
	`
	var total int
	if err := i.DB.QueryRowContext(ctx, q).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("can't count items: %w", err)
	}

	offset := (num - 1) * size
	q = `
		select id, name, sale_id, sold, created_at
		from items
		order by id
		limit $1 offset $2
	`
	rows, err := i.DB.QueryContext(ctx, q, size, offset)
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
