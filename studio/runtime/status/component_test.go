package status

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/sdk"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/catalog"
	rhandler "github.com/viant/datly/runtime/handler"
	contract "github.com/viant/forge/backend/artifactrelease"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
	"github.com/viant/xdatly/response"
)

type releaseStatusProvider struct{ db *sql.DB }

func (p releaseStatusProvider) Connector(_ context.Context, name string) (*sql.DB, error) {
	if name != "studio" {
		return nil, errors.New("unknown connector")
	}
	return p.db, nil
}

type releaseStatusBinder struct{ provider releaseStatusProvider }

func (b releaseStatusBinder) Bind(_ context.Context, target any) error {
	value := reflect.ValueOf(target)
	if value.Kind() == reflect.Pointer && value.Elem().Kind() == reflect.Struct {
		field := value.Elem().FieldByName("Authorization")
		if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
			field.SetString("Bearer verified-test")
		}
	}
	return nil
}
func (b releaseStatusBinder) Lookup(_ context.Context, key xhandler.ValueKey) (any, bool, error) {
	if key == rhandler.ConnectorCapabilityKey {
		return b.provider, true, nil
	}
	return nil, false, nil
}

type releaseStatusSession struct{ binder releaseStatusBinder }

func (s releaseStatusSession) Binder() xhandler.Binder   { return s.binder }
func (s releaseStatusSession) Response() response.Writer { return nil }

type releaseStatusManager struct{}

func (releaseStatusManager) List(context.Context, contract.ListInput) (contract.ListOutput, error) {
	return contract.ListOutput{}, nil
}
func (releaseStatusManager) Candidates(context.Context, contract.CandidatesInput) (contract.CandidatesOutput, error) {
	return contract.CandidatesOutput{}, nil
}
func (releaseStatusManager) Publish(context.Context, contract.PublishInput) (contract.Result, error) {
	return contract.Result{}, nil
}
func (releaseStatusManager) Promote(context.Context, contract.PromoteInput) (contract.Result, error) {
	return contract.Result{}, nil
}
func (releaseStatusManager) History(context.Context, contract.HistoryInput) (contract.HistoryOutput, error) {
	return contract.HistoryOutput{}, nil
}

var _ sdkrelease.Manager = releaseStatusManager{}

func TestNativeRuntimeStatusAdvertisesOnlySelectedReadyReleaseManager(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "native_release_status", "studio")
	first, second := namespaceaccess.ID("owner", "ar"), namespaceaccess.ID("owner", "other")
	for _, item := range []struct{ id, name string }{{first, "ar"}, {second, "other"}} {
		if _, err := db.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner',?,?,'active','private',TRUE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, item.id, item.name, item.name); err != nil {
			t.Fatal(err)
		}
	}
	ctx = runtimeadmin.WithClient(ctx, runtimeadmin.NewInProcess(nil, func(ctx context.Context) (*sdk.RuntimeHost, error) {
		selected, _ := sdk.NamespaceSelectionFromContext(ctx)
		return &sdk.RuntimeHost{NamespaceID: selected, Status: "ready", MCPURL: "http://127.0.0.1:12345/mcp"}, nil
	}, nil))
	defer catalog.ConfigureReleaseManagementResolver(nil)
	resolved := 0
	catalog.ConfigureReleaseManagementResolver(func(_ context.Context, selected string) (sdkrelease.Manager, error) {
		resolved++
		if selected == first {
			return releaseStatusManager{}, nil
		}
		return nil, errors.New("release unavailable")
	})
	session := releaseStatusSession{binder: releaseStatusBinder{provider: releaseStatusProvider{db}}}
	claims := &jwt.Claims{}
	claims.Subject = "owner"
	verified := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "owner"}}
	for _, item := range []struct {
		id   string
		want bool
	}{{first, true}, {second, false}} {
		out := &Output{}
		if err := NewStatus().Exec(ctx, session, &Input{NamespaceId: &item.id, Jwt: claims, Auth: verified}, out); err != nil {
			t.Fatal(err)
		}
		if out.Response == nil || out.Response.ReleasesAvailable != item.want {
			t.Fatalf("namespace %s release status: %+v", item.id, out.Response)
		}
	}
	if resolved != 2 {
		t.Fatalf("selected release resolver called %d times", resolved)
	}
	outsider := &jwt.Claims{}
	outsider.Subject = "other"
	if err := NewStatus().Exec(ctx, session, &Input{NamespaceId: &first, Jwt: outsider, Auth: &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "other"}}}, &Output{}); err == nil {
		t.Fatal("private namespace status leaked to outsider")
	}
	if resolved != 2 {
		t.Fatal("denied principal reached release resolver")
	}
	catalog.ConfigureReleaseManagementResolver(nil)
	out := &Output{}
	if err := NewStatus().Exec(ctx, session, &Input{NamespaceId: &first, Jwt: claims, Auth: verified}, out); err != nil || out.Response.ReleasesAvailable {
		t.Fatalf("disabled manager advertised: %+v %v", out.Response, err)
	}
}
