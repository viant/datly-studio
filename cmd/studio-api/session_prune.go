package main

import (
	"context"
	"log"
	"time"
)

type expiredSessionPruner interface {
	DeleteExpired(context.Context, time.Time) error
}

func startSessionPruner(ctx context.Context, store expiredSessionPruner, interval time.Duration) <-chan struct{} {
	done := make(chan struct{})
	if interval <= 0 {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			pruneCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			err := store.DeleteExpired(pruneCtx, time.Now().UTC())
			cancel()
			if err != nil && ctx.Err() == nil {
				log.Printf("Studio BFF session cleanup: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return done
}
