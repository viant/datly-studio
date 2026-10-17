// studio is the default single-process Studio host.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/viant/datly-studio/app/hostconfig"
	"github.com/viant/datly-studio/app/studiohost"
	_ "github.com/viant/datly-studio/examples/predicates"
)

func main() {
	flags := flag.NewFlagSet("studio", flag.ContinueOnError)
	configPath := flags.String("config", "studio.yaml", "shared Studio configuration")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return
		}
		os.Exit(2)
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: studio --config studio.yaml")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	config, err := hostconfig.Load(ctx, *configPath)
	if err == nil {
		err = studiohost.Serve(ctx, config, studiohost.Options{Diagnostics: os.Stderr, Ready: func(e studiohost.Endpoints) {
			fmt.Fprintf(os.Stdout, "Studio ready: HTTP=%s adminMCP=%s namespaceMCP=%s pid=%d\n", e.HTTP, e.AdminMCP, e.NamespaceMCP, os.Getpid())
		}})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
