// Package connectorinit applies explicit, server-owned connection setup that
// cannot be represented in a DSN. It never accepts browser-authored SQL.
package connectorinit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type options struct {
	SQLiteAttachments map[string]string `json:"sqliteAttachments"`
	Pool              *poolOptions      `json:"pool"`
}

// Configure applies explicit pool limits and driver-specific connection setup.
func Configure(ctx context.Context, db *sql.DB, driver string, raw json.RawMessage) error {
	if db == nil || len(raw) == 0 || string(raw) == "{}" {
		return nil
	}
	var config options
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("decode connector options: %w", err)
	}
	if err := configurePool(db, config.Pool); err != nil {
		return err
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(driver)), "sqlite") {
		return nil
	}
	if len(config.SQLiteAttachments) == 0 {
		return nil
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	names := make([]string, 0, len(config.SQLiteAttachments))
	for name := range config.SQLiteAttachments {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !validIdentifier(name) {
			return fmt.Errorf("invalid SQLite attachment name %q", name)
		}
		location := strings.TrimSpace(config.SQLiteAttachments[name])
		if location == "" {
			return fmt.Errorf("SQLite attachment %q has no location", name)
		}
		if _, err := db.ExecContext(ctx, `ATTACH DATABASE ? AS "`+name+`"`, location); err != nil {
			return fmt.Errorf("attach SQLite database %q: %w", name, err)
		}
	}
	return nil
}

func validIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || index > 0 && character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}
