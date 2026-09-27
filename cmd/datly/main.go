package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/bigquery"
	_ "github.com/viant/datly-studio/internal/dependencylink"
	"github.com/viant/datly/cmd/command"
	_ "github.com/viant/sqlx/metadata/product/bigquery"
	_ "github.com/viant/sqlx/metadata/product/mysql"
	_ "github.com/viant/sqlx/metadata/product/pg"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	_ "modernc.org/sqlite"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(os.Args) > 1 && os.Args[1] == "transcribe" {
		os.Exit(runTranscribe(ctx, os.Args[1:], os.Stdout, os.Stderr))
	}
	if len(os.Args) > 1 && os.Args[1] == "link" {
		os.Exit(runLink(ctx, os.Args[1:], os.Stdout, os.Stderr))
	}
	os.Exit((command.Service{}).Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
