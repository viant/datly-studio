package change_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/studio/namespaces/change"
	"github.com/viant/datly-studio/studio/namespaces/store_insert"
	"github.com/viant/datly-studio/studio/namespaces/store_write"
	xhandler "github.com/viant/xdatly/handler"
)

func TestNamespaceHooksNotifyOnlyConfirmedCommits(t *testing.T) {
	for _, state := range []xhandler.TransactionState{xhandler.TransactionCommitted, xhandler.TransactionRolledBack, xhandler.TransactionCallerPending, xhandler.TransactionCommitUnknown, xhandler.TransactionNone} {
		for name, finalize := range map[string]func(context.Context, xhandler.Outcome) error{
			"insert": func(ctx context.Context, outcome xhandler.Outcome) error {
				return new(store_insert.NamespaceInsertRules).Finalize(ctx, nil, &store_insert.Output{Data: []*store_insert.StoredNamespace{{NamespaceId: "owned-id", OwnerId: "owner", Name: "example"}}}, outcome)
			},
			"update": func(ctx context.Context, outcome xhandler.Outcome) error {
				return new(store_write.NamespaceStoreRules).Finalize(ctx, nil, &store_write.Output{Data: []*store_write.StoredNamespace{{NamespaceId: "owned-id", OwnerId: "owner", Name: "example"}}}, outcome)
			},
		} {
			t.Run(name+"/"+string(state), func(t *testing.T) {
				calls := 0
				ctx := change.WithObserver(context.Background(), change.ObserverFunc(func(_ context.Context, identities []change.Identity) error {
					calls++
					require.Equal(t, []change.Identity{{NamespaceID: "owned-id", OwnerID: "owner", Name: "example"}}, identities)
					return nil
				}))
				outcome := xhandler.Outcome{Transactions: []xhandler.TransactionOutcome{{State: state}}}
				require.NoError(t, finalize(ctx, outcome))
				if state == xhandler.TransactionCommitted {
					require.Equal(t, 1, calls)
				} else {
					require.Zero(t, calls)
				}
				outcome.Error = errors.New("mutation failed")
				require.NoError(t, finalize(ctx, outcome))
				if state == xhandler.TransactionCommitted {
					require.Equal(t, 1, calls)
				} else {
					require.Zero(t, calls)
				}
			})
		}
	}
}

func TestNamespaceObserverIsOptionalAndPreservesPostCommitFailure(t *testing.T) {
	identities := []change.Identity{{NamespaceID: "owned-id", OwnerID: "owner", Name: "example"}}
	require.NoError(t, change.Notify(context.Background(), identities))
	cause := errors.New("refresh failed after commit")
	ctx := change.WithObserver(context.Background(), change.ObserverFunc(func(context.Context, []change.Identity) error { return cause }))
	require.ErrorIs(t, change.Notify(ctx, identities), cause)
	require.NoError(t, change.Notify(ctx, nil))
}
