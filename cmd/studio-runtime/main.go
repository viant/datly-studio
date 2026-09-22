package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/bigquery"
	"github.com/viant/datly-studio/runtime/host"
	_ "modernc.org/sqlite"
)

func main() {
	configPath := flag.String("conf", "datly-runtime.yaml", "dynamic Datly runtime configuration")
	flag.Parse()
	config, err := host.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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
