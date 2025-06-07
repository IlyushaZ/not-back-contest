package database

import (
	"database/sql"
	"fmt"
)

type TxFunc func(*sql.Tx) error

func WithTx(db *sql.DB, fn TxFunc) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("can't begin tx: %w", err)
	}

	defer func() {
		p := recover()
		switch {
		case p != nil:
			_ = tx.Rollback()
			panic(p)

		case err != nil:
			rbErr := tx.Rollback()
			if rbErr != nil {
				err = fmt.Errorf("can't rollback tx: %w. original error: %w", rbErr, err)
			}

		default:
			err = tx.Commit()
			if err != nil {
				err = fmt.Errorf("can't commit tx: %w", err)
			}
		}
	}()

	err = fn(tx)
	return
}
