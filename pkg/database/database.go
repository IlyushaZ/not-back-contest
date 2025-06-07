package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrNotFound = errors.New("record not found")
)

func New(addr, database, user, password string) (db *sql.DB, close func() error, err error) {
	url := fmt.Sprintf("postgres://%s:%s@%s/%s", user, password, addr, database)

	db, err = sql.Open("pgx", url)
	if err != nil {
		return nil, nil, err
	}

	// these params are set assuming that max_connections are set to 200-250
	db.SetMaxOpenConns(150)
	db.SetMaxIdleConns(75)
	db.SetConnMaxLifetime(15 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, nil, err
	}

	return db, db.Close, nil
}

func mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
