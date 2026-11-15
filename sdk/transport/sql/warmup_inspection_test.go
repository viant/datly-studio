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
	"github.com/viant/datly/spec"
)

func TestInspectionEffectiveWarmup(t *testing.T) {
	f := newImportFixture(t)
	const scope = "example.org/inspection/records"
	if _, err := f.db.ExecContext(f.ctx, "UPDATE connectors SET dsn_template=':memory:'; UPDATE components SET component_scope=?,component_name='reader' WHERE id=?", scope, f.report.ID); err != nil {
		t.Fatal(err)
	}
	for _, policy := range []string{"publisher", "viewer"} {
		publish := policy == "publisher"
		if _, err := f.db.ExecContext(f.ctx, "INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,can_run,can_publish,can_use_dql) VALUES(?,'user',?,1,0,?,0)", f.report.ID, policy, publish); err != nil {
			t.Fatal(err)
		}
	}
	one, zero := 1, 0
	cache := t.TempDir()
	engine := preview.Dynamic{StudioDB: f.db, ModulePath: "example.org/inspection", ReaderCaches: map[string]*spec.CacheSettings{
		"storage": {Enabled: true, Location: cache, TTL: "1m"},
		"profile": {Warmup: &spec.CacheWarmupSettings{Name: "profile", IndexColumn: "ID", MaxCases: &one, Limit: &zero, Cases: []*spec.CacheWarmupCase{{Set: []*spec.CacheWarmupParam{{Name: "Tenant", Values: []string{"PRIVATE_CASE_CANARY"}, ExcludeDefault: true}}}}}},
	}, Dependencies: func(context.Context, *spec.Component) (*preview.DependencyRuntime, error) {
		t.Fatal("inspection invoked runtime dependencies")
		return nil, nil
	}}
	f.transport.Authorizer = AuthorizerFunc(func(ctx context.Context, r AuthorizationRequest) error {
		p, _ := sdk.PrincipalFromContext(ctx)
		if r.Permission == "publish" && p.Subject == "viewer" {
			return errors.New("denied")
		}
		return nil
	})
	f.transport.Warmup = engine
	publisher := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "publisher"})
	for _, scenario := range []string{"named", "empty", "unresolved"} {
		t.Run(scenario, func(t *testing.T) {
			projection := ""
			if scenario != "empty" {
				profile := "profile"
				if scenario == "unresolved" {
					profile = "PRIVATE_UNRESOLVED_CANARY"
				}
				projection = fmt.Sprintf(",use_cache(r,'storage'),cache_warmup(r,'%s')", profile)
			}
			dql := fmt.Sprintf("#package('%s')\n#setting($_ = $connector('source'))\n#setting($_ = $route('/records','GET'))\n#define($_ = $Rows<[]*Row>(output/view))\nSELECT r.*,type(r,'Row')%s FROM (SELECT 1 AS ID) r", scope, projection)
			v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: dql})
			if err != nil {
				t.Fatal(err)
			}
			got, err := f.client.Versions().Inspect(publisher, f.report.ID, v.VersionNo)
			if err != nil {
				t.Fatal(err)
			}
			m := got.EffectiveWarmup
			if m == nil || m.ReportID != f.report.ID || m.VersionNo != v.VersionNo || m.SourceRevision != v.SourceRevision {
				t.Fatalf("metadata identity: %+v", got)
			}
			raw, _ := json.Marshal(got)
			if got.DQL != "" || got.Capabilities.CanUseDQL || strings.Contains(string(raw), "CANARY") || strings.Contains(string(raw), cache) {
				t.Fatalf("metadata/source leak: %s", raw)
			}
			if scenario == "unresolved" {
				if m.Status != "unavailable" || m.Targets != nil {
					t.Fatalf("failed resolution became empty: %+v", m)
				}
				found := false
				for _, d := range got.Diagnostics {
					found = found || d.Code == "warmup_inspection_unavailable" && d.Message != ""
				}
				if !found {
					t.Fatal("safe diagnostic missing")
				}
			} else {
				if m.Status != "resolved" || m.Targets == nil {
					t.Fatalf("not resolved: %+v", m)
				}
				if scenario == "empty" {
					if len(*m.Targets) != 0 || !strings.Contains(string(raw), `"targets":[]`) {
						t.Fatalf("not an explicit empty list: %s", raw)
					}
				} else {
					if len(*m.Targets) != 1 {
						t.Fatal("wrong target count")
					}
					target := (*m.Targets)[0]
					if target.PolicyName != "profile" || target.MaxCases == nil || *target.MaxCases != 1 || target.RowLimit == nil || *target.RowLimit != 0 || len(target.Cases) != 1 || target.Cases[0].Dimensions[0].ValueCount != 1 || !target.Cases[0].Dimensions[0].ExcludeDefault {
						t.Fatalf("native projection: %+v", target)
					}
				}
			}
			viewer := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "viewer"})
			viewed, err := f.client.Versions().Inspect(viewer, f.report.ID, v.VersionNo)
			if err != nil || viewed.EffectiveWarmup != nil {
				t.Fatalf("view permission granted warmup metadata: %v %+v", err, viewed)
			}
			t.Run("exact warmup authorization", func(t *testing.T) {
				previous := f.transport.Authorizer
				defer func() { f.transport.Authorizer = previous }()
				calls := 0
				f.transport.Authorizer = AuthorizerFunc(func(ctx context.Context, r AuthorizationRequest) error {
					if r.Operation == sdk.OperationVersionWarmup {
						calls++
						return errors.New("denied")
					}
					return nil
				})
				denied, err := f.client.Versions().Inspect(publisher, f.report.ID, v.VersionNo)
				if err != nil || calls != 1 || denied.EffectiveWarmup != nil {
					t.Fatalf("warmup authorization bypassed: %v %d %+v", err, calls, denied)
				}
			})
		})
	}
}
