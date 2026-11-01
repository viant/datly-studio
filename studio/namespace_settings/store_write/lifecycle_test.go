package store_write

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/studio/namespaces/change"
	xhandler "github.com/viant/xdatly/handler"
)

func TestSettingsNotifyHostOnlyAfterConfirmedCommit(t *testing.T) {
	for _, state := range []xhandler.TransactionState{xhandler.TransactionCommitted, xhandler.TransactionRolledBack, xhandler.TransactionCallerPending, xhandler.TransactionCommitUnknown, xhandler.TransactionNone} {
		t.Run(string(state), func(t *testing.T) {
			calls := 0
			ctx := change.WithObserver(context.Background(), change.ObserverFunc(func(_ context.Context, identities []change.Identity) error {
				calls++
				require.Equal(t, []change.Identity{{NamespaceID: namespaceaccess.ID("owner", "example"), OwnerID: "owner", Name: "example", SettingsChanged: true, Revision: 2}}, identities)
				return nil
			}))
			revision := int64(2)
			output := &Output{Data: []*StoredNamespace{{OwnerId: "owner", Name: "example", Etag: &revision}}}
			outcome := xhandler.Outcome{Transactions: []xhandler.TransactionOutcome{{State: state}}}
			require.NoError(t, new(NamespaceSettingsRules).Finalize(ctx, nil, output, outcome))
			want := 0
			if state == xhandler.TransactionCommitted {
				want = 1
			}
			require.Equal(t, want, calls)
			outcome.Error = errors.New("write failed")
			require.NoError(t, new(NamespaceSettingsRules).Finalize(ctx, nil, output, outcome))
			require.Equal(t, want, calls)
		})
	}
}
