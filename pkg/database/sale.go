package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IlyushaZ/not-back-contest/pkg/model"
)

type SaleRepository interface {
	GetPage(ctx context.Context, num, size int) ([]model.Sale, int, error)
}

type SaleDatabase struct {
	DB *sql.DB
}

func (sd *SaleDatabase) GetPage(ctx context.Context, num, size int) ([]model.Sale, int, error) {
	q := `
		select count(*) from sales
	`
	var total int
	if err := sd.DB.QueryRowContext(ctx, q).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("can't count sales: %w", err)
	}

	offset := (num - 1) * size
	q = `
		select id, created_at, start_at, end_at
		from sales
		order by created_at desc
		limit $1 offset $2
	`
	rows, err := sd.DB.QueryContext(ctx, q, size, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("can't query sales: %w", err)
	}
	defer rows.Close()

	ss := make([]model.Sale, 0, size)
	for rows.Next() {
		var s model.Sale
		if err := rows.Scan(&s.ID, &s.CreatedAt, &s.StartAt, &s.EndAt); err != nil {
			return nil, 0, fmt.Errorf("can't scan sale: %w", err)
		}

		ss = append(ss, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating over sales: %w", err)
	}

	return ss, total, nil
}
