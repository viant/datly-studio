package connectors

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/sdk"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/transcribe/column"
)

// SecretResolver resolves server-held connector references. It is never given
// a browser or model-supplied DSN; deployments may replace the default SCY
// resolver with their own trusted secret provider.
type SecretResolver interface {
	Resolve(context.Context, string, string) (string, error)
}

// Opened is one borrowable Datly SQL connector set. Call Close after the
// exact-version runtime finishes; no application CRUD occurs in this package.
type Opened struct {
	SQL         *dsql.SQLComponent
	Connections column.Connections
	opened      []*sql.DB
}

// Open resolves an explicit primary connector and any other exact named
// dependencies. The caller selects configurations from its trusted catalog;
// duplicate, missing, inactive, or incomplete entries fail before execution.
func Open(ctx context.Context, primary string, configured []*sdk.Connector, secrets SecretResolver) (*Opened, error) {
	if primary == "" || len(configured) == 0 {
		return nil, fmt.Errorf("primary connector and configurations are required")
	}
	result := &Opened{Connections: column.Connections{}}
	completed := false
	defer func() {
		if !completed {
			_ = result.Close()
		}
	}()
	for _, item := range configured {
		if item == nil || item.Name == "" || item.Status != "active" || item.Driver == "" {
			return nil, fmt.Errorf("active connector name and driver are required")
		}
		if result.Connections[item.Name] != nil {
			return nil, fmt.Errorf("duplicate connector %q", item.Name)
		}
		dsn, err := connectorsecret.Resolve(ctx, secrets, item.DSNTemplate, item.SecretRef)
		if err != nil || strings.TrimSpace(dsn) == "" {
			return nil, fmt.Errorf("connector %q DSN could not be resolved", item.Name)
		}
		driver := strings.ToLower(strings.TrimSpace(item.Driver))
		if driver == "sqlite3" {
			driver = "sqlite"
		}
		db, err := sql.Open(driver, dsn)
		if err != nil {
			return nil, fmt.Errorf("open connector %q: %w", item.Name, err)
		}
		result.opened = append(result.opened, db)
		if err := connectorinit.Configure(ctx, db, driver, item.Options); err != nil {
			return nil, fmt.Errorf("configure connector %q: %w", item.Name, err)
		}
		if err := db.PingContext(ctx); err != nil {
			return nil, fmt.Errorf("ping connector %q: %w", item.Name, err)
		}
		result.Connections[item.Name] = db
		if item.Name == primary {
			result.SQL = &dsql.SQLComponent{DB: db}
		}
	}
	if result.SQL == nil {
		return nil, fmt.Errorf("primary connector %q is unavailable", primary)
	}
	for name, db := range result.Connections {
		if err := result.SQL.RegisterConnector(name, db); err != nil {
			return nil, err
		}
	}
	completed = true
	return result, nil
}

func (o *Opened) Close() error {
	if o == nil {
		return nil
	}
	var err error
	for _, db := range o.opened {
		err = errors.Join(err, db.Close())
	}
	o.opened = nil
	return err
}
