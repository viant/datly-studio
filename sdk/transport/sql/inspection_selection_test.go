package sqltransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/sourcevalidation"
	"github.com/viant/datly/spec"
)

func TestInspectionMetadataSelection(t *testing.T) {
	f := newImportFixture(t)
	const scope = "example.org/selection/records"
	if _, err := f.db.ExecContext(f.ctx, "UPDATE connectors SET dsn_template=':memory:'; UPDATE components SET component_scope=?,component_name='reader' WHERE id=?", scope, f.report.ID); err != nil {
		t.Fatal(err)
	}
	versions := []*sdk.ReportVersion{}
	for _, policy := range []string{"profile", "missing-policy"} {
		dql := fmt.Sprintf("#package('%s')\n#setting($_ = $connector('source'))\n#setting($_ = $route('/records','GET'))\n#define($_ = $ID<int>(query/id).Optional())\n#define($_ = $Rows<[]*Row>(output/view))\nSELECT r.*,type(r,'Row'),use_cache(r,'storage'),cache_warmup(r,'%s') FROM (SELECT 1 AS ID) r", scope, policy)
		v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: dql})
		if err != nil {
			t.Fatal(err)
		}
		versions = append(versions, v)
	}
	loads := 0
	engine := &preview.Dynamic{StudioDB: f.db, ModulePath: "example.org/selection", SourceValidator: func(ctx context.Context, input sourcevalidation.Input) error { loads++; return nil }, ReaderCaches: map[string]*spec.CacheSettings{
		"storage": {Enabled: true, Location: t.TempDir(), TTL: "1m"}, "profile": {Warmup: &spec.CacheWarmupSettings{Name: "profile", IndexColumn: "ID"}},
	}, Dependencies: func(context.Context, *spec.Component) (*preview.DependencyRuntime, error) {
		t.Fatal("inspection executed dependencies")
		return nil, nil
	}}
	f.transport.Preview = engine
	f.transport.Warmup = engine
	f.transport.ContractInspector = engine
	for _, tc := range []struct {
		name                  string
		version               int
		input, warmup         *bool
		wantInput, wantWarmup bool
		loads                 int
	}{
		{"initial", 0, boolPointer(false), boolPointer(false), false, false, 0},
		{"preview", 0, boolPointer(true), boolPointer(false), true, false, 1},
		{"warmup", 0, boolPointer(false), boolPointer(true), false, true, 1},
		{"both", 0, boolPointer(true), boolPointer(true), true, true, 1},
		{"legacy", 0, nil, nil, true, true, 1},
		{"warmup-failure", 1, boolPointer(true), boolPointer(true), true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loads = 0
			v := versions[tc.version]
			request := map[string]any{"reportId": f.report.ID, "versionNo": v.VersionNo, "discoverColumns": false}
			if tc.input != nil {
				request["includePreviewInput"] = *tc.input
			}
			if tc.warmup != nil {
				request["includeEffectiveWarmup"] = *tc.warmup
			}
			var got sdk.ReaderInspection
			if err := f.transport.Invoke(f.ctx, sdk.OperationVersionInspect, request, &got); err != nil {
				t.Fatal(err)
			}
			if loads != tc.loads || (got.PreviewInput != nil) != tc.wantInput || (got.EffectiveWarmup != nil) != tc.wantWarmup {
				t.Fatalf("loads=%d inspection=%+v", loads, got)
			}
			if got.PreviewInput != nil && (got.PreviewInput.VersionNo != v.VersionNo || got.PreviewInput.SourceRevision != v.SourceRevision) {
				t.Fatal("preview snapshot mismatch")
			}
			if got.EffectiveWarmup != nil {
				if got.EffectiveWarmup.VersionNo != v.VersionNo || got.EffectiveWarmup.SourceRevision != v.SourceRevision {
					t.Fatal("warmup snapshot mismatch")
				}
				if tc.version == 1 {
					if got.EffectiveWarmup.Status != "unavailable" || got.EffectiveWarmup.Targets != nil {
						t.Fatal("failure became empty")
					}
				} else if got.EffectiveWarmup.Status != "resolved" || got.EffectiveWarmup.Targets == nil || len(*got.EffectiveWarmup.Targets) != 1 {
					t.Fatalf("missing target: %+v", got.EffectiveWarmup)
				}
			}
			codes := []string{}
			for _, d := range got.Diagnostics {
				if strings.HasSuffix(d.Code, "_unavailable") {
					codes = append(codes, d.Code)
				}
			}
			if tc.version == 1 {
				if len(codes) != 1 || codes[0] != "warmup_inspection_unavailable" {
					t.Fatalf("independent diagnostics: %v", codes)
				}
			} else if len(codes) != 0 {
				t.Fatalf("unexpected diagnostics: %v", codes)
			}
			stored, err := f.client.Versions().Get(f.ctx, f.report.ID, v.VersionNo)
			if err != nil || stored.SourceRevision != v.SourceRevision || stored.AuthoredDQL != v.AuthoredDQL {
				t.Fatalf("inspection changed source: %v", err)
			}
		})
	}
	t.Run("independent permissions", func(t *testing.T) {
		previous := f.transport.Authorizer
		defer func() { f.transport.Authorizer = previous }()
		for _, denied := range []string{sdk.OperationPreviewExecute, sdk.OperationVersionWarmup} {
			f.transport.Authorizer = AuthorizerFunc(func(ctx context.Context, request AuthorizationRequest) error {
				if request.Operation == denied {
					return errors.New("denied")
				}
				return nil
			})
			var got sdk.ReaderInspection
			if err := f.transport.Invoke(f.ctx, sdk.OperationVersionInspect, map[string]any{"reportId": f.report.ID, "versionNo": versions[0].VersionNo, "discoverColumns": false}, &got); err != nil {
				t.Fatal(err)
			}
			if (got.PreviewInput != nil) != (denied != sdk.OperationPreviewExecute) || (got.EffectiveWarmup != nil) != (denied != sdk.OperationVersionWarmup) {
				raw, _ := json.Marshal(got)
				t.Fatalf("authorization mixed projections: %s", raw)
			}
		}
	})
}

func boolPointer(value bool) *bool { return &value }
