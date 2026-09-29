package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/bigquery"
	"github.com/viant/datly-studio/runtime/host"
	"github.com/viant/datly-studio/runtime/namespacemcp"
	_ "github.com/viant/sqlx/metadata/product/bigquery"
	_ "github.com/viant/sqlx/metadata/product/mysql"
	_ "github.com/viant/sqlx/metadata/product/pg"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	_ "modernc.org/sqlite"
)

func main() {
	configPath := flag.String("conf", "datly-runtime.yaml", "dynamic Datly runtime configuration")
	namespaceMode := flag.Bool("namespace-mcp", false, "serve persisted namespace MCP endpoints without a global MCP listener")
	reconcileInterval := flag.Duration("namespace-reconcile-interval", 2*time.Second, "namespace MCP configuration polling interval")
	flag.Parse()
	config, err := host.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *namespaceMode {
		if err := runNamespaces(ctx, *config, *reconcileInterval); err != nil {
			log.Fatal(err)
		}
		return
	}
	service, err := host.New(ctx, *config)
	if err != nil {
		log.Fatal(err)
	}
	if err = service.Start(ctx); err != nil {
		log.Fatal(err)
	}
	log.Printf("dynamic Datly HTTP listening on %s; MCP listening on %s", config.HTTP.Address, config.MCP.Address)
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = service.Close(shutdown); err != nil {
		log.Print(err)
	}
}

func runNamespaces(ctx context.Context, config host.Config, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("namespace reconciliation interval must be positive")
	}
	group, err := host.NewNamespaceGroup(ctx, config)
	if err != nil {
		return err
	}
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := group.Close(shutdown); err != nil {
			log.Print("namespace runtime shutdown failed")
		}
	}()
	if err := group.StartAdmin(config.HTTP.Address, config.Admin.Token); err != nil {
		return err
	}
	previous := map[string]string{}
	return group.Run(ctx, interval, func(changes []namespacemcp.Change, err error) {
		if err != nil && len(changes) == 0 {
			log.Print("namespace MCP configuration could not be read")
			return
		}
		for _, change := range changes {
			address := ""
			if change.Endpoint != nil {
				address = change.Endpoint.Address
			}
			state := change.Status + ":" + address
			if previous[change.NamespaceID] == state {
				continue
			}
			previous[change.NamespaceID] = state
			log.Printf("namespace %s MCP %s %s", change.NamespaceID, change.Status, address)
		}
	})
}
