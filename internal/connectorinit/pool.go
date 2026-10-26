package connectorinit

import (
	"database/sql"
	"fmt"
)

// Pointers distinguish omitted settings from database/sql's explicit zero values.
type poolOptions struct {
	MaxOpenConns *int `json:"maxOpenConns"`
	MaxIdleConns *int `json:"maxIdleConns"`
}

func configurePool(db *sql.DB, pool *poolOptions) error {
	if pool == nil {
		return nil
	}
	for name, value := range map[string]*int{"maxOpenConns": pool.MaxOpenConns, "maxIdleConns": pool.MaxIdleConns} {
		if value != nil && *value < 0 {
			return fmt.Errorf("connector pool %s must be nonnegative", name)
		}
	}
	if pool.MaxOpenConns != nil && pool.MaxIdleConns != nil && *pool.MaxOpenConns > 0 && *pool.MaxIdleConns > *pool.MaxOpenConns {
		return fmt.Errorf("connector pool maxIdleConns exceeds maxOpenConns")
	}
	if pool.MaxOpenConns != nil {
		db.SetMaxOpenConns(*pool.MaxOpenConns)
	}
	if pool.MaxIdleConns != nil {
		db.SetMaxIdleConns(*pool.MaxIdleConns)
	}
	return nil
}
