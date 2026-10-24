package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestPreviewRechecksOnlyWhenReleaseGrantWasUsed(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"reportId": "reader", "versionNo": 7, "input": sdk.PreviewInput{Limit: 1}}
	for _, test := range []struct {
		name      string
		release   bool
		revoke    bool
		wantCalls int
	}{
		{name: "legacy", wantCalls: 1},
		{name: "release", release: true, wantCalls: 2},
		{name: "release_revoked", release: true, revoke: true, wantCalls: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			authorizer := AuthorizerFunc(func(ctx context.Context, _ AuthorizationRequest) error {
				calls++
				if test.release {
					MarkReleaseUse(ctx)
				}
				if test.revoke && calls == 2 {
					return errors.New("revoked")
				}
				return nil
			})
			transport := &Transport{DB: db, Authorizer: authorizer, Preview: previewStub{}}
			var result sdk.PreviewResult
			err := transport.Invoke(ctx, sdk.OperationPreviewExecute, input, &result)
			if test.revoke && err == nil || !test.revoke && err != nil {
				t.Fatalf("preview err=%v", err)
			}
			if calls != test.wantCalls {
				t.Fatalf("authorization calls=%d want=%d", calls, test.wantCalls)
			}
		})
	}
}
