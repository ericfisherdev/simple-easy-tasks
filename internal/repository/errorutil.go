// Package repository provides data access interfaces and implementations.
package repository

import (
	"database/sql"
	"errors"
	"strings"
)

// ErrNotFound is a sentinel error for not found conditions
var ErrNotFound = errors.New("not found")

// IsNotFound checks if an error represents a "not found" condition.
// It uses errors.Is for proper error checking and falls back to legacy
// string comparison for compatibility with older error handling.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	// Check for standard SQL no rows error
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}

	// Check for our sentinel error
	if errors.Is(err, ErrNotFound) {
		return true
	}

	// Legacy string check fallback for compatibility
	errStr := err.Error()
	return strings.Contains(errStr, "no rows in result set") ||
		strings.Contains(errStr, "sql: no rows in result set")
}
