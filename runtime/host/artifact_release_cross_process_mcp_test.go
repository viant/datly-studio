package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/forge/backend/artifactcohort"
	artifactcache "github.com/viant/forge/backend/artifactcohort/cache"
	contract "github.com/viant/forge/backend/artifactrelease"
	forgeservice "github.com/viant/forge/backend/provider"
	windowprotocol "github.com/viant/forge/backend/window/contract"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

// Two independently projected producer catalogs share durable SQL. A commits
// the release; B's background watcher changes its own manifest and sends the
// standard notification over B's real authenticated MCP endpoint to a replica.
func TestRemoteReleaseCommitReachesOtherProducerMCPConsumer(t *testing.T) {
	identity := newStockOAuthFixture(t)
	ownerToken := identity.token(t, identity.private, "owner", []string{"reader"}, time.Now().Add(time.Minute))
	var token atomic.Value
	token.Store(ownerToken)
	rootBody := closedWindowForCohort(t, "cross-process")
	root := artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: 1, Hash: artifactcohort.Hash(rootBody)}
	artifact := &artifactcohort.Artifact{Root: root, Payload: rootBody}
	policyResource := authz.Resource{Tenant: "team", Kind: "policy", ID: "readers", Version: "logical"}
	policyRef := authz.NamedReference{Resource: policyResource, Revision: 1}
	policy := authz.Document{Resource: policyResource, Revision: 1, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: "Readers", Policies: map[string]authz.Policy{
		"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}},
		"execute":  {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}},
	}}}}}
	var sourceAvailable atomic.Bool
	sourceAvailable.Store(true)
	var restoredB, notifiedB, checkedB atomic.Int32
	publisherB := artifactcohort.NewPublisher()
	if _, err := publisherB.ReplaceCatalog(context.Background(), nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	catalogB := NewPublishedArtifactCatalog(publisherB)
	provider := &forgeservice.PrimitiveProvider{Host: forgeservice.PrimitiveHostFuncs{CatalogFunc: func(context.Context, *windowprotocol.CatalogInput) (*windowprotocol.Catalog, error) {
		return &windowprotocol.Catalog{ContractVersion: windowprotocol.Version, CatalogRevision: "fixture"}, nil
	}}, Authority: forgeservice.PrimitiveAuthorityFuncs{AuthenticateFunc: func(context.Context) (string, error) { return "fixture", nil }, AuthorizeFunc: func(context.Context, string, string, string, string) error { return nil }}}
	var publisherA *sdkrelease.Controller
	testDynamicHost(t, true, false, dynamicHostExtension{skipDefaultChecks: true,
		configure: func(config *Config) {
			config.Access = &ResourceAccessConfig{Tenant: "team", Provider: identity.provider}
			config.ForgeProvider = provider
			config.ArtifactCohorts = catalogB
			config.ReleaseSyncInterval = 100 * time.Millisecond
			config.AuthorizeArtifactHost = func(ctx context.Context) error {
				principal, err := identity.provider.ResolvePrincipal(ctx)
				if err != nil || principal.Facts.Subject != "owner" {
					return authz.ErrDenied
				}
				return nil
			}
		},
		prepare: func(host *Service) {
			ctx := context.Background()
			writerDB, err := sql.Open("sqlite", host.config.Studio.DSN)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = writerDB.Close() })
			scope := sqlstore.Scope{Tenant: "team", Account: "21", Workspace: "workspace", Namespace: "demo"}
			makeController := func(db *sql.DB, p *artifactcohort.Publisher, second bool) *sdkrelease.Controller {
				store, err := sqlstore.New(db, scope, "verified-owner")
				if err != nil {
					t.Fatal(err)
				}
				controller, err := sdkrelease.NewController(sdkrelease.ControllerConfig{
					Scope: scope, Store: store, Publisher: p,
					VerifyScope: func(ctx context.Context) (sqlstore.Scope, error) {
						principal, err := identity.provider.ResolvePrincipal(ctx)
						if err != nil || principal.Facts.Subject != "owner" {
							return sqlstore.Scope{}, authz.ErrDenied
						}
						return scope, nil
					},
					Authorize:  func(context.Context, string) error { return nil },
					Candidates: func(context.Context) ([]contract.Candidate, error) { return []contract.Candidate{{Root: root}}, nil },
					Prepare: func(context.Context, []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
						return []*artifactcohort.Artifact{artifact}, nil
					},
					CompileBindings: func(context.Context, contract.Bundle, []*artifactcohort.Artifact) ([]sqlstore.ComponentBinding, error) {
						return nil, nil
					},
					RestoreArtifacts: func(context.Context, []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
						if second {
							restoredB.Add(1)
						}
						return []*artifactcohort.Artifact{artifact}, nil
					},
					AvailableRoot: func(context.Context, artifactcohort.RootRef) error {
						if second {
							checkedB.Add(1)
						}
						if !sourceAvailable.Load() {
							return artifactcohort.ErrDenied
						}
						return nil
					},
					ValidateArtifact: func(context.Context, *artifactcohort.Artifact) error { return nil },
					NamedDocuments: func(context.Context, []authz.NamedReference) ([]authz.Document, error) {
						return []authz.Document{policy}, nil
					},
					Fence:         func(context.Context, []authz.NamedDependency) error { return nil },
					AdmitAudience: func(context.Context, contract.Channel, authz.NamedReference) (bool, error) { return true, nil },
					Notify: func(ctx context.Context) {
						if second {
							notifiedB.Add(1)
							catalogB.NotifyListChanged(ctx)
						}
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				return controller
			}
			fresh := artifactcohort.NewPublisher()
			if _, err := fresh.ReplaceCatalog(ctx, nil, nil, 0); err != nil {
				t.Fatal(err)
			}
			publisherA = makeController(writerDB, fresh, false)
			if err := publisherA.Recover(ctx); err != nil {
				t.Fatal(err)
			}
			host.config.ReleaseManagement = makeController(host.studio, publisherB, true)
		},
		verify: func(_ *Service, address string) {
			ctx := context.Background()
			notifications := artifactcohort.NewNotificationHandler(ctx, nil)
			defer notifications.Close()
			transport, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithHandler(mcpclient.NewHandler(notifications)), streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &token}}), streamable.WithProtocolVersion(schema.LegacyProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
				var wire struct {
					Method string
					Params struct{ Name string }
				}
				if err := json.Unmarshal(body, &wire); err != nil {
					return err
				}
				headers.Set(schema.HeaderMethod, wire.Method)
				if wire.Method == schema.MethodToolsCall {
					headers.Set(schema.HeaderName, wire.Params.Name)
				}
				return nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			client := mcpclient.New("other-producer-consumer", "1", transport, mcpclient.WithProtocolVersion(schema.LegacyProtocolVersion))
			defer client.Close()
			if _, err = client.Initialize(ctx); err != nil {
				t.Fatal(err)
			}
			counted := &countedCohortToolClient{next: client, calls: map[string]int{}, fetchByURI: map[string]int{}}
			caller := artifactcohort.MCPCaller{Client: counted}
			backingDir := filepath.Join(t.TempDir(), "blobs")
			indexDir := filepath.Join(t.TempDir(), "indexes")
			backing, err := artifactcache.NewFileBacking(backingDir)
			if err != nil {
				t.Fatal(err)
			}
			store, err := artifactcache.New(artifactcache.Config{SizeMB: 1, MaxEntries: 8, MaxEntryBytes: 64 << 10, Backing: backing})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			replicaConfig := artifactcohort.Config{Enabled: true, Source: "other-producer", Tenant: "team", Account: "21", Caller: caller, Facts: identity.provider, SessionKey: func(context.Context) (string, error) { return "owner-session", nil }, Store: store, ManifestStore: artifactcache.FileManifestStore{Directory: indexDir}}
			replica, err := artifactcohort.NewReplica(replicaConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err = notifications.AttachReplica(replica); err != nil {
				t.Fatal(err)
			}
			ownerCtx := oauth.WithBearer(ctx, ownerToken)
			if err = replica.Sync(ownerCtx); err != nil {
				t.Fatal(err)
			}
			if _, err = publisherA.Publish(ownerCtx, contract.PublishInput{Namespace: "demo", Name: "standard", Policies: []authz.NamedReference{policyRef}, Roots: []artifactcohort.RootRef{root}}); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if err = replica.Admit(ownerCtx); err == nil {
					if view, openErr := replica.Open(ownerCtx, root.URI); openErr == nil && view.Ref() == root {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
			}
			if view, openErr := replica.Open(ownerCtx, root.URI); openErr != nil || view.Ref() != root {
				t.Fatalf("other producer notification did not open released root: %v", openErr)
			}
			if notifiedB.Load() != 1 || restoredB.Load() != 1 || counted.fetchCount(root.URI) != 1 {
				t.Fatalf("cross-process notification=%d restore=%d fetch=%d", notifiedB.Load(), restoredB.Load(), counted.fetchCount(root.URI))
			}
			priorChecks := checkedB.Load()
			deadline = time.Now().Add(2 * time.Second)
			for checkedB.Load() < priorChecks+2 && time.Now().Before(deadline) {
				time.Sleep(5 * time.Millisecond)
			}
			if checkedB.Load() < priorChecks+2 {
				t.Fatal("background worker made no unchanged passes")
			}
			if notifiedB.Load() != 1 || restoredB.Load() != 1 || counted.fetchCount(root.URI) != 1 {
				t.Fatal("unchanged background passes rebuilt or downloaded bytes")
			}
			// A fresh consumer object confirms the producer hash over MCP and
			// hydrates from persistent backing without another artifact/fetch.
			reopenedBacking, err := artifactcache.NewFileBacking(backingDir)
			if err != nil {
				t.Fatal(err)
			}
			reopenedStore, err := artifactcache.New(artifactcache.Config{SizeMB: 1, MaxEntries: 8, MaxEntryBytes: 64 << 10, Backing: reopenedBacking})
			if err != nil {
				t.Fatal(err)
			}
			defer reopenedStore.Close()
			restartConfig := replicaConfig
			restartConfig.Store = reopenedStore
			restartConfig.ManifestStore = artifactcache.FileManifestStore{Directory: indexDir}
			restarted, err := artifactcohort.NewReplica(restartConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err = restarted.Sync(ownerCtx); err != nil {
				t.Fatal(err)
			}
			if err = restarted.Admit(ownerCtx); err != nil {
				t.Fatal(err)
			}
			if _, err = restarted.Open(ownerCtx, root.URI); err != nil {
				t.Fatal(err)
			}
			if counted.fetchCount(root.URI) != 1 || reopenedStore.Hydrations() != 1 {
				t.Fatalf("unchanged consumer restart fetched=%d hydrated=%d", counted.fetchCount(root.URI), reopenedStore.Hydrations())
			}
			sourceAvailable.Store(false)
			deadline = time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if _, openErr := replica.Open(ownerCtx, root.URI); openErr != nil && notifiedB.Load() >= 2 {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			if _, openErr := replica.Open(ownerCtx, root.URI); openErr == nil || notifiedB.Load() != 2 {
				t.Fatalf("other producer did not withdraw root: %v notifications=%d", openErr, notifiedB.Load())
			}
			if counted.fetchCount(root.URI) != 1 {
				t.Fatal("withdrawal downloaded unchanged artifact")
			}
		},
	})
}
