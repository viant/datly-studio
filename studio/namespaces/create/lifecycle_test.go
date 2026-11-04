package create

import (
	"context"
	"errors"
	"github.com/viant/datly-studio/studio/namespaces/change"
	h "github.com/viant/xdatly/handler"
	"testing"
)

func TestNamespaceCreateNotifiesOnlyCommittedOutcome(t *testing.T) {
	for _, state := range []h.TransactionState{h.TransactionCommitted, h.TransactionRolledBack, h.TransactionCallerPending, h.TransactionCommitUnknown, h.TransactionNone} {
		t.Run(string(state), func(t *testing.T) {
			calls := 0
			ctx := change.WithObserver(context.Background(), change.ObserverFunc(func(_ context.Context, ids []change.Identity) error {
				calls++
				if len(ids) != 1 || ids[0].NamespaceID != "id" || ids[0].OwnerID != "owner" || ids[0].Name != "example" {
					t.Fatal("wrong committed identity")
				}
				return nil
			}))
			err := new(NamespaceCreateRules).Finalize(ctx, nil, &NamespaceCreateOutput{Data: &NamespaceRecord{NamespaceId: "id", OwnerId: "owner", Name: "example"}}, h.Outcome{Transactions: []h.TransactionOutcome{{State: state}}})
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if state == h.TransactionCommitted {
				want = 1
			}
			if calls != want {
				t.Fatalf("notifications=%d want=%d", calls, want)
			}
		})
	}
	expected := errors.New("refresh failed")
	ctx := change.WithObserver(context.Background(), change.ObserverFunc(func(context.Context, []change.Identity) error { return expected }))
	err := new(NamespaceCreateRules).Finalize(ctx, nil, &NamespaceCreateOutput{Data: &NamespaceRecord{NamespaceId: "id", OwnerId: "owner", Name: "example"}}, h.Outcome{Transactions: []h.TransactionOutcome{{State: h.TransactionCommitted}}})
	if !errors.Is(err, expected) {
		t.Fatal("post-commit refresh error lost")
	}
}
