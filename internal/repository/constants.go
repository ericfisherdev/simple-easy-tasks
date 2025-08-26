package repository

import (
	"database/sql"
	"errors"
)

// IsNoRows checks if the error is a "no rows" error from SQL
func IsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
