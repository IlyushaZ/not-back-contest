package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/config"
	"github.com/IlyushaZ/not-back-contest/pkg/database"
	"github.com/IlyushaZ/not-back-contest/pkg/model"
)

var cfg = config.New()

var ErrSaleExists = errors.New("sale exists")

// words used for generating items' names
var (
	categories = []string{"Electronics", "Clothing", "Books", "Home", "Sports", "Beauty", "Toys", "Food", "Health", "Garden"}
	adjectives = []string{"Premium", "Deluxe", "Ultra", "Pro", "Smart", "Classic", "Modern", "Vintage", "Luxury", "Budget"}
	items      = []string{"Phone", "Laptop", "Watch", "Headphones", "Camera", "Tablet", "Speaker", "Keyboard", "Mouse", "Monitor"}
)

// this should run by cron every hour in 1 instance. I should have done the workers which can run in multiple instances and synchronize, but im too lazy.
func main() {
	t0 := time.Now()
	defer func() { log.Printf("Items generated. Elapsed: %s", time.Since(t0)) }()

	db, closeDB, err := database.New(cfg.PostgresAddr, cfg.PostgresDB, cfg.PostgresUser, cfg.PostgresPassword)
	if err != nil {
		log.Fatalf("### Can't init database: %v", err)
	}
	defer closeDB()

	if err := generate(db); err != nil {
		log.Fatalf("### Can't generate items: %v", err)
	}
}

func generate(db *sql.DB) error {
	now := time.Now()

	for i := 0; i < cfg.SalesCount; i++ {
		start := now.Truncate(time.Hour)
		end := start.Add(model.SaleDuration)

		err := database.WithTx(db, func(tx *sql.Tx) error {
			const saleExists = `
				select exists (
					select 1
					from sales
					where start_at = $1 and end_at = $2
				) as exists
			`

			var exists bool

			if err := tx.QueryRow(saleExists, start, end).Scan(&exists); err != nil {
				return fmt.Errorf("can't check if sale exists: %w", err)
			}

			if exists {
				return ErrSaleExists
			}

			const insertSale = `
				insert into sales (start_at, end_at, created_at)
				values ($1, $2, $3)
				returning id
			`

			var saleID int

			if err := tx.QueryRow(insertSale, start, end, now).Scan(&saleID); err != nil {
				return fmt.Errorf("can't insert sale: %w", err)
			}

			stmt, err := tx.Prepare(`insert into items (sale_id, name, created_at, sale_start, sale_end) values ($1, $2, $3, $4, $5)`)
			if err != nil {
				return fmt.Errorf("can't prepare stmt for inserting item: %w", err)
			}

			for j := 0; j < cfg.ItemsPerSale; j++ {
				item := generateItem(saleID, start, end, now)
				if _, err := stmt.Exec(item.SaleID, item.Name, now, item.SaleStart, item.SaleEnd); err != nil {
					return fmt.Errorf("can't insert item: %w", err)
				}

				if (j+1)%100 == 0 {
					log.Printf("Inserted %d items for sale #%d\n", j+1, i+1)
				}
			}

			return nil
		})
		if err != nil {
			if errors.Is(err, ErrSaleExists) {
				slog.Warn("Sale with such start and end already exists",
					slog.Time("start", start),
					slog.Time("end", end),
				)
				continue
			}

			return fmt.Errorf("can't add sale to database: %w", err)
		}

		now = now.Add(time.Hour) // next sale will be for the next item

		log.Printf("Sale #%d added\n", i+1)
	}

	return nil
}

func generateItem(saleID int, saleStart, saleEnd, createdAt time.Time) *model.Item {
	adj := adjectives[rand.Intn(len(adjectives))]
	category := categories[rand.Intn(len(categories))]
	item := items[rand.Intn(len(items))]

	return &model.Item{
		Base:      model.Base{CreatedAt: createdAt},
		SaleID:    saleID,
		SaleStart: saleStart,
		SaleEnd:   saleEnd,
		Name:      fmt.Sprintf("%s %s %s", adj, category, item),
	}
}
