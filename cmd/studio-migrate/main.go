package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/store/sql/fixture"
	"github.com/viant/datly-studio/store/sql/migrate"
	_ "modernc.org/sqlite"
)

func main() {
	var (
		command = flag.String("command", "init-studio", "migration command: init-studio | up | down-to | version | seed-sample")
		dsn     = flag.String("dsn", "", "database DSN; for sqlite a file path or sqlite URI")
		target  = flag.Int("target", 0, "target schema version for down-to")
	)
	flag.Parse()

	ctx := context.Background()
	db, err := openDB(strings.TrimSpace(*dsn))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		log.Fatalf("enable foreign keys: %v", err)
	}

	switch strings.TrimSpace(*command) {
	case "init-studio":
		if err := applyCanonical(ctx, db); err != nil {
			log.Fatalf("initialize Studio schema: %v", err)
		}
		if err := schema.ApplySQLite(ctx, db, "studio_seed"); err != nil {
			log.Fatalf("initialize Studio fixtures: %v", err)
		}
		fmt.Println("initialized canonical Studio schema and fixtures")
	case "up":
		if err := applyCanonical(ctx, db); err != nil {
			log.Fatalf("migrate up: %v", err)
		}
		version, err := schema.SQLiteVersion(ctx, db)
		if err != nil {
			log.Fatalf("read schema version: %v", err)
		}
		fmt.Printf("schema version: %d\n", version)
	case "down-to":
		if *target != 0 && *target != schema.CanonicalVersion {
			log.Fatalf("unsupported canonical target version: %d", *target)
		}
		if *target == 0 {
			if err := schema.DropSQLite(ctx, db); err != nil {
				log.Fatalf("migrate down-to %d: %v", *target, err)
			}
		}
		version, err := schema.SQLiteVersion(ctx, db)
		if err != nil {
			log.Fatalf("read schema version: %v", err)
		}
		fmt.Printf("schema version: %d\n", version)
	case "version":
		version, err := schema.SQLiteVersion(ctx, db)
		if err != nil {
			log.Fatalf("read schema version: %v", err)
		}
		fmt.Printf("schema version: %d\n", version)
	case "seed-sample":
		if err := applyCanonical(ctx, db); err != nil {
			log.Fatalf("migrate up before seeding: %v", err)
		}
		if err := clearCanonicalSeed(ctx, db); err != nil {
			log.Fatalf("clear sample catalog: %v", err)
		}
		seed, err := fixture.SeedSampleCatalog(ctx, db, time.Now().UTC())
		if err != nil {
			log.Fatalf("seed sample catalog: %v", err)
		}
		fmt.Printf("seeded connectors: %s, %s\n", seed.PrimaryConnectorName, seed.SecondaryConnectorName)
		fmt.Printf("seeded studio fixtures: %s, %s\n", seed.DraftSlug, seed.PublishedSlug)
	default:
		flag.Usage()
		log.Fatalf("unsupported command: %s", *command)
	}
}

func applyCanonical(ctx context.Context, db *sql.DB) error {
	service, err := migrate.New()
	if err != nil {
		return err
	}
	return service.Up(ctx, db)
}

func clearCanonicalSeed(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DELETE FROM report_acl; DELETE FROM report_publications; DELETE FROM report_mcp_exposures; DELETE FROM report_cube_configs; DELETE FROM report_predicates; DELETE FROM report_parameters; DELETE FROM report_fields; DELETE FROM report_views; DELETE FROM report_versions; DELETE FROM reports; DELETE FROM runtime_generations; DELETE FROM connectors`)
	return err
}

func openDB(rawDSN string) (*sql.DB, error) {
	dsn := rawDSN
	if dsn == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolve working directory: %w", err)
		}
		dsn = filepath.Join(wd, "reporting.sqlite")
	}
	dsn = normalizeSQLiteDSN(dsn)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func normalizeSQLiteDSN(value string) string {
	switch {
	case strings.HasPrefix(value, "file:"):
		return value
	case strings.Contains(value, "?"):
		return "file:" + value
	default:
		return "file:" + value
	}
}
