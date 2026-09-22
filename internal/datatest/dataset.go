package datatest

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Dataset is an ordered, JSON-authored fixture. Table order is significant so
// parents can be inserted before rows that reference them.
type Dataset struct {
	Tables []Table `json:"tables"`
}

// HydrationPhase validates and inserts one JSON dataset transactionally.
type HydrationPhase struct {
	JSON []byte
}

func (p HydrationPhase) Apply(ctx context.Context, db *sql.DB) error {
	dataset, err := DecodeDataset(p.JSON)
	if err != nil {
		return err
	}
	return Hydrate(ctx, db, dataset.Tables...)
}

// DecodeDataset preserves JSON numbers and validates the fixture envelope.
func DecodeDataset(data []byte) (*Dataset, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	result := &Dataset{}
	if err := decoder.Decode(result); err != nil {
		return nil, fmt.Errorf("decode hydration dataset: %w", err)
	}
	if len(result.Tables) == 0 {
		return nil, fmt.Errorf("hydration dataset requires at least one table")
	}
	seen := map[string]bool{}
	for index := range result.Tables {
		table := &result.Tables[index]
		table.Name = strings.TrimSpace(table.Name)
		if table.Name == "" {
			return nil, fmt.Errorf("hydration table %d name is required", index)
		}
		if seen[table.Name] {
			return nil, fmt.Errorf("hydration table %q is duplicated", table.Name)
		}
		seen[table.Name] = true
	}
	if decoder.More() {
		return nil, fmt.Errorf("hydration dataset contains trailing JSON values")
	}
	return result, nil
}

type fixtureColumn struct {
	name       string
	notNull    bool
	primaryKey bool
	hasDefault bool
}

func validateFixtureTable(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, table Table) (map[string]fixtureColumn, error) {
	quoted, err := quoteIdentifier(table.Name)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+quoted+")")
	if err != nil {
		return nil, fmt.Errorf("inspect fixture table %s: %w", table.Name, err)
	}
	defer rows.Close()
	columns := map[string]fixtureColumn{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, dataType string
		var defaultValue sql.NullString
		if err = rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns[name] = fixtureColumn{name: name, notNull: notNull != 0, primaryKey: primaryKey != 0, hasDefault: defaultValue.Valid}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("hydrate table %q does not exist", table.Name)
	}
	for rowIndex, row := range table.Rows {
		for name, value := range row {
			column, ok := columns[name]
			if !ok {
				return nil, fmt.Errorf("hydrate %s row %d has unknown column %q", table.Name, rowIndex, name)
			}
			if value == nil && column.notNull {
				return nil, fmt.Errorf("hydrate %s row %d column %q is required", table.Name, rowIndex, name)
			}
		}
		for name, column := range columns {
			if !column.notNull || column.hasDefault {
				continue
			}
			if _, ok := row[name]; !ok {
				return nil, fmt.Errorf("hydrate %s row %d is missing required column %q", table.Name, rowIndex, name)
			}
		}
	}
	return columns, nil
}

func normalizeFixtureValue(value any) (any, error) {
	switch actual := value.(type) {
	case json.Number:
		if integer, err := strconv.ParseInt(string(actual), 10, 64); err == nil {
			return integer, nil
		}
		decimal, err := strconv.ParseFloat(string(actual), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON number %q", actual)
		}
		return decimal, nil
	case map[string]any, []any:
		encoded, err := json.Marshal(actual)
		if err != nil {
			return nil, err
		}
		return string(encoded), nil
	default:
		return value, nil
	}
}

// RowsFromJSON decodes an expected ordered row array for AssertRows.
func RowsFromJSON(data []byte) ([]Row, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var result []Row
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("decode expected rows: %w", err)
	}
	for i := range result {
		for key, value := range result[i] {
			normalized, err := normalizeFixtureValue(value)
			if err != nil {
				return nil, fmt.Errorf("expected row %d field %s: %w", i, key, err)
			}
			result[i][key] = normalized
		}
	}
	return result, nil
}
