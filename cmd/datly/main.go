package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/viant/datly-studio/app/datlycmd"
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
	os.Exit(datlycmd.Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
