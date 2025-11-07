package search

import (
	"fmt"

	"github.com/spideyz0r/fh/pkg/storage"
)

// Search queries the database and returns matching entries.
func Search(db *storage.DB, query string, limit int) ([]*storage.HistoryEntry, error) {
	filters := storage.QueryFilters{
		Search: query,
		Limit:  limit,
	}

	entries, err := db.Query(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}

	return entries, nil
}

// All returns all history entries (most recent first).
func All(db *storage.DB, limit int) ([]*storage.HistoryEntry, error) {
	return Search(db, "", limit)
}

// WithFilters searches with custom filters.
func WithFilters(db *storage.DB, filters storage.QueryFilters) ([]*storage.HistoryEntry, error) {
	entries, err := db.Query(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}

	return entries, nil
}
