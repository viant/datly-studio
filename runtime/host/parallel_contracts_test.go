package host

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContractPreparationOverlapsIndependentReadersAndRetainsOrder(t *testing.T) {
	definitions := []definition{{reportID: "0"}, {reportID: "1"}, {reportID: "2"}, {reportID: "3"}}
	started := make(chan string, len(definitions))
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	type result struct {
		values []int
		err    error
	}
	done := make(chan result, 1)
	go func() {
		values, err := compileContracts(context.Background(), definitions, func(ctx context.Context, item definition) (int, error) {
			started <- item.reportID
			select {
			case <-release:
				return strconv.Atoi(item.reportID)
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		})
		done <- result{values, err}
	}()
	for range definitions {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("independent contract reads did not start together")
		}
	}
	close(release)
	got := <-done
	require.NoError(t, got.err)
	require.Equal(t, []int{0, 1, 2, 3}, got.values)
}

func TestContractPreparationCancelsPeersAfterError(t *testing.T) {
	want := errors.New("reader unavailable")
	definitions := []definition{{reportID: "bad"}, {reportID: "other"}}
	_, err := compileContracts(context.Background(), definitions, func(ctx context.Context, item definition) (int, error) {
		if item.reportID == "bad" {
			return 0, want
		}
		<-ctx.Done()
		return 0, ctx.Err()
	})
	require.ErrorIs(t, err, want)
}
