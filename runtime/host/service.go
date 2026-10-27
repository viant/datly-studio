package host

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/datly-studio/internal/namespacepolicy"
	xcodec "github.com/viant/xdatly/codec"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	access "github.com/viant/authz"
	"github.com/viant/datly-studio/audit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/internal/resourceartifact"
	studiors "github.com/viant/datly-studio/runtime/resources"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	studiohost "github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/application"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	gateway "github.com/viant/datly/gateway/http"
	"github.com/viant/datly/mcp"
	mcpserver "github.com/viant/datly/mcp/server"
	"github.com/viant/datly/report"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	"github.com/viant/datly/typecatalog"
	coreresource "github.com/viant/forge/backend/resourcehost"
	"github.com/viant/forge/backend/types"
	skillformat "github.com/viant/mcp-protocol/extension/skills"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/scy/auth/jwt/verifier"
	"github.com/viant/sqlx/io/read/cache/aerospike"
	xresponse "github.com/viant/xdatly/response"
)

type verifiedClaimsKey struct{}

type Service struct {
	codecs                 xcodec.Factory
	windowExecutionProof   types.WindowTargetProof
	metadataDefinitions    *exactDefinitionReader
	metadataResources      *studiors.Loader
	windowPrimitives       *coreresource.LocalProvider
	windowPrimitiveSource  *stockPrimitiveSource
	windowPrimitivePolicy  *stockWindowPolicy
	resourceAccess         *access.Service
	namespacePolicyReader  *namespacepolicy.Reader
	config                 Config
	studio                 *sql.DB
	connectorStore         *activeConnectorStore
	definitionStore        *publishedDefinitionStore
	compiledArtifacts      *compiledArtifactStore
	compileRuntimeContract func(context.Context, *transcribe.Source) (*transcribe.RuntimeContract, error)
	runAccessStore         *runAccessStore
	manager                *application.Manager
	servers                []*http.Server
	listeners              []net.Listener
	verifier               *verifier.Service
	providerVerifiers      map[string]*verifier.Service
	secrets                connectorsecret.Resolver
	mu                     sync.Mutex
	releaseSyncController  *sdkrelease.Controller
	releaseSyncCancel      context.CancelFunc
	releaseSyncDone        chan struct{}
}

func New(ctx context.Context, config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	codecs, err := runtimeCodecs(ctx, config)
	if err != nil {
		return nil, err
	}
	studio, err := sql.Open(config.Studio.Driver, config.Studio.DSN)
	if err != nil {
		return nil, err
	}
	if err = studio.PingContext(ctx); err != nil {
		studio.Close()
		return nil, err
	}
	types, err := (studiohost.Config{PredicatePackages: config.PredicatePackages, CodecPackages: config.CodecPackages}).RuntimeTypes()
	if err != nil {
		studio.Close()
		return nil, err
	}
	manager, err := application.New(types)
	if err != nil {
		studio.Close()
		return nil, err
	}
	definitionStore, err := newPublishedDefinitionStore(studio)
	if err != nil {
		_ = manager.Shutdown(context.Background())
		_ = studio.Close()
		return nil, fmt.Errorf("runtime definition store: %w", err)
	}
	runAccessStore, err := newRunAccessStore(studio)
	if err != nil {
		_ = definitionStore.Close(context.Background())
		_ = manager.Shutdown(context.Background())
		_ = studio.Close()
		return nil, fmt.Errorf("runtime run access store: %w", err)
	}
	compiledArtifacts, err := newCompiledArtifactStore(studio)
	if err != nil {
		_ = runAccessStore.Close(context.Background())
		_ = definitionStore.Close(context.Background())
		_ = manager.Shutdown(context.Background())
		_ = studio.Close()
		return nil, fmt.Errorf("runtime compiled artifact store: %w", err)
	}
	result := &Service{codecs: codecs, config: config, studio: studio, manager: manager, connectorStore: &activeConnectorStore{db: studio}, definitionStore: definitionStore, runAccessStore: runAccessStore, compiledArtifacts: compiledArtifacts}
	if err = result.initResourceAccess(); err != nil {
		_ = compiledArtifacts.Close(context.Background())
		_ = runAccessStore.Close(context.Background())
		_ = definitionStore.Close(context.Background())
		_ = manager.Shutdown(context.Background())
		_ = studio.Close()
		return nil, err
	}
	if err = result.initStockForgeProvider(ctx); err != nil {
		_ = compiledArtifacts.Close(context.Background())
		_ = runAccessStore.Close(context.Background())
		_ = definitionStore.Close(context.Background())
		_ = manager.Shutdown(context.Background())
		_ = studio.Close()
		return nil, err
	}
	result.providerVerifiers = map[string]*verifier.Service{}
	for name, provider := range config.Authentication.Providers {
		service := verifier.New(&verifier.Config{CertURL: provider.CertURL})
		if err = service.Init(ctx); err != nil {
			_ = compiledArtifacts.Close(context.Background())
			_ = runAccessStore.Close(context.Background())
			_ = definitionStore.Close(context.Background())
			_ = manager.Shutdown(context.Background())
			_ = studio.Close()
			return nil, fmt.Errorf("runtime provider %s: %w", name, err)
		}
		result.providerVerifiers[name] = service
	}
	if config.Access == nil && !strings.EqualFold(config.Authentication.DefaultMode, "public") {
		result.verifier = verifier.New(&verifier.Config{CertURL: config.Authentication.CertURL})
		if err = result.verifier.Init(ctx); err != nil {
			_ = compiledArtifacts.Close(context.Background())
			_ = runAccessStore.Close(context.Background())
			_ = definitionStore.Close(context.Background())
			_ = manager.Shutdown(context.Background())
			_ = studio.Close()
			return nil, err
		}
	}
	return result, nil
}

func (s *Service) Reload(ctx context.Context, generation int64) error {
	if generation <= 0 {
		return s.reload(ctx, nil)
	}
	return s.reload(ctx, &generation)
}

func (s *Service) reload(ctx context.Context, candidate *int64) error {
	revision := s.manager.Revision() + 1
	if candidate != nil && *candidate > 0 && uint64(*candidate) > revision {
		revision = uint64(*candidate)
	}
	return s.manager.Reload(ctx, application.Request{Revision: revision, Compile: func(ctx context.Context, seed *typecatalog.Catalog) (*application.Build, error) {
		return s.compile(ctx, seed, candidate)
	}})
}

func (s *Service) compile(ctx context.Context, seed *typecatalog.Catalog, candidate *int64) (*application.Build, error) {
	definitions, err := s.definitions(ctx, candidate)
	if err != nil {
		return nil, err
	}
	build, err := s.compileDefinitions(ctx, seed, definitions)
	if err != nil {
		return nil, err
	}
	return s.extendNamespaceBuild(ctx, build)
}

func (s *Service) extendNamespaceBuild(ctx context.Context, build *application.Build) (*application.Build, error) {
	if s.config.NamespaceID == "" {
		return build, nil
	}
	extend := s.config.NamespaceBuildExtensions[s.config.NamespaceID]
	if extend == nil && s.config.NamespaceBuildResolver != nil {
		var err error
		extend, err = s.config.NamespaceBuildResolver(ctx, s.config.NamespaceID)
		if err != nil {
			if build != nil && build.Shutdown != nil {
				err = errors.Join(err, build.Shutdown(context.Background()))
			}
			return nil, fmt.Errorf("resolve namespace %s build extension: %w", s.config.NamespaceID, err)
		}
	}
	if extend == nil {
		return build, nil
	}
	if err := extend(ctx, build); err != nil {
		if build.Shutdown != nil {
			err = errors.Join(err, build.Shutdown(context.Background()))
		}
		return nil, fmt.Errorf("compose namespace %s: %w", s.config.NamespaceID, err)
	}
	return build, nil
}

func (s *Service) compileDefinitions(ctx context.Context, seed *typecatalog.Catalog, definitions []definition, inspect ...func(definition, *registry.RegisteredComponent, *studiors.Loaded) error) (_ *application.Build, err error) {
	started := time.Now()
	defer func() {
		slog.InfoContext(ctx, "Studio namespace compilation", "namespaceId", s.config.NamespaceID, "components", len(definitions), "elapsedMs", float64(time.Since(started))/float64(time.Millisecond), "failed", err != nil)
	}()

	phaseStarted := time.Now()
	types, err := seed.Clone()
	if err != nil {
		return nil, err
	}
	compiledArtifactTiming(ctx, "namespace.seed.clone", time.Since(phaseStarted), 0)
	versions := make([]studiors.Version, 0, len(definitions))
	for _, definition := range definitions {
		versions = append(versions, studiors.Version{ReportID: definition.reportID, VersionNo: definition.versionNo})
	}
	phaseStarted = time.Now()
	loadedResources, err := studiors.Load(ctx, s.studio, versions)
	if err != nil {
		return nil, err
	}
	compiledArtifactTiming(ctx, "namespace.resources.load", time.Since(phaseStarted), 0)
	resources := loadedResources.Store
	registrations := make([]*registry.RegisteredComponent, 0, len(definitions))
	reportByComponent := map[spec.Key]string{}
	versionByReport := map[string]int{}
	for _, definition := range definitions {
		versionByReport[definition.reportID] = definition.versionNo
	}
	var pooled *runtimeConnectorSet
	caches := &aerospike.Pool{}
	defer func() {
		if err != nil {
			_ = caches.Close()
			if pooled != nil {
				_ = pooled.Close()
			}
		}
	}()
	phaseStarted = time.Now()
	active, activeErr := s.activeConnectors(ctx)
	if activeErr != nil {
		return nil, activeErr
	}
	compiledArtifactTiming(ctx, "namespace.connectors.read", time.Since(phaseStarted), 0)
	phaseStarted = time.Now()
	pooled, err = openRuntimeConnectorSet(ctx, definitions, active, s.secrets, s.config.Const)
	if err != nil {
		return nil, err
	}
	compiledArtifactTiming(ctx, "namespace.connectors.open", time.Since(phaseStarted), 0)
	type contractResult struct {
		sources  *runtimeSources
		contract *transcribe.RuntimeContract
		elapsed  time.Duration
	}
	compileOne := func(compileCtx context.Context, definition definition, catalog *typecatalog.Catalog) (contractResult, error) {
		started := time.Now()
		version := studiors.Version{ReportID: definition.reportID, VersionNo: definition.versionNo}
		contractResources := loadedResources.ByVersion[version]
		var cacheIdentity resourceartifact.Identity
		var cacheOwner string
		var repairDigest string
		bypassReason := "store-unavailable"
		cacheable := s.compiledArtifacts != nil
		if cacheable {
			available := make([]string, 0, len(active))
			for name := range active {
				available = append(available, name)
			}
			sort.Strings(available)
			selected := map[string]connectorDefinition{}
			for _, name := range referencedRuntimeConnectors(compileCtx, definition, available, s.config.Const) {
				item, exists := active[name]
				if !exists {
					cacheable = false
					bypassReason = "connector-unavailable"
					break
				}
				selected[name] = item
			}
			if cacheable {
				var identityErr error
				identityStarted := time.Now()
				cacheIdentity, cacheOwner, identityErr = compilerInputIdentity(s.config.NamespaceID, s.config.ModulePath, s.config.RootDir,
					definition, loadedResources.Fingerprints[version], s.config.Const.Identity(), selected, catalog)
				compiledArtifactTiming(compileCtx, "artifact.identity", time.Since(identityStarted), 0)
				if identityErr != nil {
					cacheable = false
					bypassReason = "identity-unavailable"
				}
			}
		}
		if cacheable {
			lookupStarted := time.Now()
			stored, lookupErr := s.compiledArtifacts.Get(compileCtx, definition.reportID, definition.versionNo, cacheIdentity)
			compiledArtifactTiming(compileCtx, "artifact.lookup", time.Since(lookupStarted), len(stored))
			missReason := "absent"
			if lookupErr == nil {
				var frozen transcribe.RuntimeContractArtifact
				if json.Unmarshal(stored, &frozen) == nil {
					hydrateStarted := time.Now()
					contract, hydrateErr := frozen.Materialize(catalog, contractResources)
					compiledArtifactTiming(compileCtx, "artifact.hydrate", time.Since(hydrateStarted), len(stored))
					if hydrateErr == nil && requireReaderOnlyContract(contract.Component) == nil {
						sources, openErr := pooled.sources(compileCtx, definition.connector)
						if openErr != nil {
							return contractResult{}, openErr
						}
						compiledArtifactTiming(compileCtx, "artifact.hit", time.Since(started), len(stored))
						return contractResult{sources: sources, contract: contract, elapsed: time.Since(started)}, nil
					}
					missReason = "hydrate-invalid"
				} else {
					missReason = "decode-invalid"
				}
				prior := sha256.Sum256(stored)
				repairDigest = hex.EncodeToString(prior[:])
			} else if errors.Is(lookupErr, errCompiledArtifactCorrupt) {
				missReason = "checksum-invalid"
				if len(stored) > 0 {
					prior := sha256.Sum256(stored)
					repairDigest = hex.EncodeToString(prior[:])
				}
			}
			if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) && !errors.Is(lookupErr, errCompiledArtifactCorrupt) {
				return contractResult{}, fmt.Errorf("read compiled artifact: %w", lookupErr)
			}
			compiledArtifactTimingReason(compileCtx, "artifact.miss", missReason, time.Since(lookupStarted), 0)
		} else {
			compiledArtifactTimingReason(compileCtx, "artifact.bypass", bypassReason, time.Since(started), 0)
		}
		sources, openErr := pooled.sources(compileCtx, definition.connector)
		if openErr != nil {
			return contractResult{}, openErr
		}
		source := &transcribe.Source{Const: s.config.Const, Types: catalog, Scope: definition.scope, Name: definition.name, Text: definition.dql, Connector: definition.connector, Resources: contractResources, ColumnRefiner: column.New(sources.connections)}
		var contract *transcribe.RuntimeContract
		var compileErr error
		discoveryStarted := time.Now()
		if s.compileRuntimeContract != nil {
			contract, compileErr = s.compileRuntimeContract(compileCtx, source)
		} else {
			compiler := transcribe.NewCompiler()
			if s.config.ModulePath != "" {
				contract, compileErr = compiler.RuntimeContractsInModule(compileCtx, s.config.ModulePath, source)
			} else {
				contract, compileErr = compiler.RuntimeContracts(compileCtx, s.config.RootDir, source)
			}
		}
		if compileErr != nil {
			return contractResult{}, fmt.Errorf("compile dynamic report %s: %w", definition.reportID, compileErr)
		}
		compiledArtifactTiming(compileCtx, "artifact.discovery", time.Since(discoveryStarted), 0)
		if contractErr := requireReaderOnlyContract(contract.Component); contractErr != nil {
			return contractResult{}, fmt.Errorf("compile dynamic report %s: %w", definition.reportID, contractErr)
		}
		if cacheable {
			frozen, freezeErr := contract.Freeze()
			if freezeErr == nil {
				encoded, encodeErr := json.Marshal(frozen)
				if encodeErr != nil {
					return contractResult{}, encodeErr
				}
				storeStarted := time.Now()
				var putErr error
				if repairDigest != "" {
					putErr = s.compiledArtifacts.Repair(compileCtx, cacheOwner, definition.reportID, definition.versionNo, cacheIdentity, encoded, repairDigest)
				} else {
					putErr = s.compiledArtifacts.Put(compileCtx, cacheOwner, definition.reportID, definition.versionNo, cacheIdentity, encoded)
				}
				compiledArtifactTiming(compileCtx, "artifact.store", time.Since(storeStarted), len(encoded))
				if putErr != nil && !errors.Is(putErr, errCompiledArtifactTooLarge) {
					return contractResult{}, fmt.Errorf("publish compiled artifact: %w", putErr)
				}
				if errors.Is(putErr, errCompiledArtifactTooLarge) {
					compiledArtifactTiming(compileCtx, "artifact.oversize", 0, len(encoded))
				}
			} else {
				compiledArtifactTimingReason(compileCtx, "artifact.unsupported", "freeze-unsupported", 0, 0)
			}
		}
		return contractResult{sources: sources, contract: contract, elapsed: time.Since(started)}, nil
	}
	phaseStarted = time.Now()
	prepared, parallelErr := compileContracts(ctx, definitions, func(compileCtx context.Context, definition definition) (contractResult, error) {
		catalog, cloneErr := seed.Clone()
		if cloneErr != nil {
			return contractResult{}, cloneErr
		}
		return compileOne(compileCtx, definition, catalog)
	})
	compiledArtifactTiming(ctx, "namespace.contracts.stage", time.Since(phaseStarted), 0)
	if parallelErr != nil {
		// A reader may import a type generated by an earlier reader. Preserve
		// that ordered contract when independent preparation cannot resolve it.
		slog.WarnContext(ctx, "Studio parallel contract preparation retried in order", "namespaceId", s.config.NamespaceID)
	}
	var artifactsElapsed, registrationsElapsed time.Duration
	for i, definition := range definitions {
		var current contractResult
		if parallelErr != nil {
			current, err = compileOne(ctx, definition, types)
			if err != nil {
				return nil, err
			}
		} else {
			current = prepared[i]
		}
		sources, contract := current.sources, current.contract
		if mergeErr := mergeTypes(types, contract.Types); mergeErr != nil {
			err = mergeErr
			return nil, err
		}
		contractFinished := time.Now()
		compilation, buildErr := report.NewProjectCompiler(report.ProjectConfig{Types: contract.Types}).CompileArtifacts([]bootstrap.ArtifactInput{{Const: s.config.Const, Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources, CodecFactory: s.codecs}})
		if buildErr != nil {
			err = buildErr
			return nil, err
		}
		artifactsFinished := time.Now()
		artifactsElapsed += artifactsFinished.Sub(contractFinished)
		compiledRegistrations, registerErr := compilation.RuntimeComponents(ctx, report.RuntimeConfigureFunc(func(_ context.Context, artifact *report.ComponentArtifact) (report.RuntimeCapabilities, error) {
			if artifact.IsReport() {
				return report.RuntimeCapabilities{}, nil
			}
			compiled := artifact.ReaderCompilation()
			readerConfig := bootstrap.ReaderRuntimeConfig{SQL: sources.sql, CacheSettings: s.config.ReaderCaches, Aerospike: caches}
			reader, readerErr := compiled.NewExecution(readerConfig)
			if readerErr != nil {
				return report.RuntimeCapabilities{}, readerErr
			}
			providers, providerErr := compiled.NewProviders(readerConfig)
			if providerErr != nil {
				return report.RuntimeCapabilities{}, providerErr
			}
			return report.RuntimeCapabilities{Reader: reader, Providers: providers}, nil
		}))
		if registerErr != nil {
			err = registerErr
			return nil, err
		}
		registrationsElapsed += time.Since(artifactsFinished)
		slog.InfoContext(ctx, "Studio component compilation", "namespaceId", s.config.NamespaceID, "componentId", definition.reportID,
			"contractMs", float64(current.elapsed)/float64(time.Millisecond),
			"artifactsMs", float64(artifactsFinished.Sub(contractFinished))/float64(time.Millisecond),
			"runtimeMs", float64(time.Since(artifactsFinished))/float64(time.Millisecond))
		registrations = append(registrations, compiledRegistrations...)
		for _, registered := range compiledRegistrations {
			for _, capture := range inspect {
				if err = capture(definition, registered, loadedResources); err != nil {
					return nil, err
				}
			}
			reportByComponent[registered.Component.Key] = definition.reportID
		}
	}
	compiledArtifactTiming(ctx, "namespace.artifacts.compile", artifactsElapsed, 0)
	compiledArtifactTiming(ctx, "namespace.runtime.register", registrationsElapsed, 0)
	if err = validateSkillToolReferences(ctx, s.studio, versions, registrations); err != nil {
		return nil, err
	}
	// Published components bind their access context as an ordinary Datly
	// component dependency; the context component is server-owned and
	// registered in the same generation at that component's concrete route.
	authorizer := s.publishedAuthorizer(versionByReport)
	accessContexts, bindsAccessContext, contextErr := authorizer.accessContexts(registrations, reportByComponent, versionByReport)
	if contextErr != nil {
		err = contextErr
		return nil, err
	}
	registrations = append(registrations, accessContexts...)
	authorizeTarget := func(ctx context.Context, target dexec.ComponentTarget) error {
		if err := authorizer.authorizeNamespace(ctx); err != nil {
			return err
		}
		reportID := reportByComponent[target.Component]
		if reportID == "" {
			return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("published component authorization is unavailable")}
		}
		if s.config.Access != nil {
			resource := access.Resource{Kind: "component", ID: reportID, Version: strconv.Itoa(versionByReport[reportID]), Tenant: s.config.Access.Tenant}
			err := authorizer.authorizeComponentExecution(ctx, resource, bindsAccessContext[reportID])
			if err != nil && admittedReleaseFor(ctx, resource) && errors.Is(err, access.ErrDenied) {
				return nil
			}
			return err
		}
		return authorizer.authorizeRun(ctx, reportID)
	}
	authorizeCatalogTool, authorizeCatalogResource, authorizeResource := authorizer.catalogAuthorizers(reportByComponent, versionByReport, loadedResources.ResourceReports)
	namespaceFolders, namespaceURIs, err := s.namespaceSkillFolders(ctx)
	if err != nil {
		return nil, err
	}
	loadedResources.Folders = append(loadedResources.Folders, namespaceFolders...)
	authorizeCatalogResource = authorizer.namespaceSkillAuthorizer(namespaceURIs, authorizeCatalogResource)
	namespaceCatalogResource := authorizeCatalogResource
	authorizeResource = func(ctx context.Context, uri string) error {
		return namespaceCatalogResource(ctx, uri, "retrieve")
	}
	connectionFolder, connectionURI, err := s.namespaceConnectionFolder(ctx)
	if err != nil {
		return nil, err
	}
	if connectionFolder != nil {
		loadedResources.Folders = append(loadedResources.Folders, *connectionFolder)
		originalRead, originalCatalog := authorizeResource, authorizeCatalogResource
		authorizeResource = func(ctx context.Context, uri string) error {
			if uri == connectionURI {
				return authorizer.authorizeNamespace(ctx)
			}
			return originalRead(ctx, uri)
		}
		authorizeCatalogResource = func(ctx context.Context, uri, action string) error {
			if uri == connectionURI {
				return authorizer.authorizeNamespace(ctx)
			}
			return originalCatalog(ctx, uri, action)
		}
	}
	cacheAdmin := func(ctx context.Context, _ *http.Request, target dexec.ComponentTarget) error {
		return authorizeCacheAdministration(ctx, target, reportByComponent, versionByReport)
	}
	httpConfig := gateway.Config{Warmup: &gateway.WarmupConfig{Timeout: 30 * time.Second, Authorize: func(context.Context, *http.Request, dexec.ComponentTarget) error {
		return errors.New("dynamic warmup administration is available through the Studio SDK")
	}, Completed: func(gateway.WarmupResult, error) {}}, CacheInvalidation: &gateway.CacheInvalidationConfig{Timeout: 5 * time.Second, Authorize: cacheAdmin}, Authorize: func(ctx context.Context, request *http.Request, target dexec.ComponentTarget) error {
		if request.URL.Path == gateway.DefaultCacheInvalidateURI || strings.HasPrefix(request.URL.Path, gateway.DefaultCacheInvalidateURI+"/") {
			return cacheAdmin(ctx, request, target)
		}
		return authorizeTarget(ctx, target)
	}}
	registrations, err = s.config.MCPToolNames.registrations(registrations)
	if err != nil {
		return nil, err
	}

	return &application.Build{
		Components: registrations,
		Types:      types,
		Resources:  resources,
		HTTP:       httpConfig,
		MCP: mcp.Config{Folders: loadedResources.Folders, AuthorizeTool: authorizeTarget, AuthorizeResource: authorizeResource,
			AuthorizeCatalogTool: authorizeCatalogTool, AuthorizeCatalogResource: authorizeCatalogResource,
			ToolMetadata: func(_ context.Context, target dexec.ComponentTarget) (map[string]interface{}, error) {
				reportID := reportByComponent[target.Component]
				version := versionByReport[reportID]
				if reportID == "" || version < 1 {
					return nil, fmt.Errorf("published MCP component has no exact source version")
				}
				identity := mcp.ToolSourceIdentity{ReportID: reportID, Version: version}
				if s.config.Access != nil {
					identity.Tenant = s.config.Access.Tenant
				}
				return map[string]interface{}{mcp.ToolSourceMetaKey: identity}, nil
			},
		},
		Version: fmt.Sprintf("dynamic-%d-%d", s.manager.Revision()+1, len(registrations)),
		Shutdown: func(context.Context) error {
			result := caches.Close()
			result = errors.Join(result, pooled.Close())
			return result
		},
	}, nil
}

func requireReaderOnlyContract(component *spec.Component) error {
	if component == nil {
		return errors.New("dynamic component contract is unavailable")
	}
	if component.Settings != nil && strings.TrimSpace(component.Settings.Mutation) != "" {
		return errors.New("dynamic components support readers only; mutation settings are not allowed")
	}
	for _, route := range component.Routes {
		if route == nil {
			continue
		}
		// POST can carry a typed reader input; PUT/PATCH/DELETE imply a writer.
		switch strings.ToUpper(strings.TrimSpace(route.Method)) {
		case http.MethodGet, http.MethodPost, http.MethodHead:
		default:
			return fmt.Errorf("dynamic components support readers only; %s %s is not allowed", route.Method, route.Path)
		}
	}
	return nil
}

// validateSkillToolReferences makes the frontmatter contract authoritative at
// publication time. UI selection is helpful, but direct SDK callers must not
// be able to publish a skill that names a non-existent tool.
func validateSkillToolReferences(ctx context.Context, db *sql.DB, versions []studiors.Version, components []*registry.RegisteredComponent) error {
	// Native skills/list and skills/get are protocol methods, not callable tools.
	// Frontmatter tool references must name an actual published component tool.
	known := map[string]bool{}
	for _, component := range components {
		if component == nil || component.Component == nil {
			continue
		}
		for _, route := range component.Component.Routes {
			if route == nil {
				continue
			}
			for _, exposure := range route.MCP {
				if exposure != nil && exposure.Kind == spec.MCPExposureTool {
					if name := exposure.Identity(component.Component, route); name != "" {
						known[name] = true
					}
				}
			}
		}
	}
	store, err := newSkillValidationStore(db)
	if err != nil {
		return err
	}
	defer store.Close(context.Background())
	for _, version := range versions {
		skills, err := store.Skills(ctx, version.ReportID, version.VersionNo)
		if err != nil {
			return err
		}
		for _, skill := range skills {
			skillID, root, namespace, folder := skill.SkillId, skill.SkillRoot, skill.Namespace, skill.RootPath
			if root == "" || root == "." {
				root = "."
			}
			resourcePath := path.Join(folder, root, "SKILL.md")
			content, err := store.Content(ctx, version.ReportID, version.VersionNo, namespace, resourcePath)
			if err != nil {
				return fmt.Errorf("skill %s content: %w", skillID, err)
			}
			frontmatter, parseErr := skillformat.Frontmatter(content)
			if parseErr != nil {
				return fmt.Errorf("skill %s frontmatter: %w", skillID, parseErr)
			}
			items, parseErr := declaredSkillTools(frontmatter)
			if parseErr != nil {
				return fmt.Errorf("skill %s: %w", skillID, parseErr)
			}
			seen := map[string]bool{}
			for _, name := range items {
				if strings.TrimSpace(name) == "" {
					return fmt.Errorf("skill %s has an invalid tool reference", skillID)
				}
				if seen[name] {
					return fmt.Errorf("skill %s repeats tool %q", skillID, name)
				}
				seen[name] = true
				if !known[name] {
					return fmt.Errorf("skill %s references MCP tool %q absent from this generation", skillID, name)
				}
			}
		}
	}
	return nil
}

func declaredSkillTools(frontmatter map[string]interface{}) ([]string, error) {
	portable, exists := frontmatter["allowed-tools"]
	if !exists {
		return nil, nil
	}
	value, ok := portable.(string)
	if !ok {
		return nil, fmt.Errorf("allowed-tools must be a string")
	}
	return strings.Fields(value), nil
}

func reportForResourceURI(prefixes map[string]string, uri string) string {
	var reportID string
	longest := -1
	for prefix, candidate := range prefixes {
		if strings.HasPrefix(uri, prefix) && len(prefix) > longest {
			reportID = candidate
			longest = len(prefix)
		}
	}
	return reportID
}

type definition struct {
	reportID, scope, name, connector, driver, dsn, secretRef, dql string
	versionNo                                                     int
}

type runtimeSources struct {
	sql         *dsql.SQLComponent
	connections column.Connections
}

func compiledArtifactTiming(ctx context.Context, phase string, elapsed time.Duration, bytes int) {
	compiledArtifactTimingReason(ctx, phase, "", elapsed, bytes)
}

func compiledArtifactTimingReason(ctx context.Context, phase, reason string, elapsed time.Duration, bytes int) {
	if os.Getenv("BACKEND_PRIMITIVE_TRACE") != "1" {
		return
	}
	slog.InfoContext(ctx, "Studio compiled artifact", "phase", phase, "reason", reason,
		"elapsedMs", float64(elapsed)/float64(time.Millisecond), "bytes", bytes)
}

func (s *Service) activeConnectors(ctx context.Context) (map[string]connectorDefinition, error) {
	return s.connectorStore.Read(ctx)
}

func (s *Service) definitions(ctx context.Context, candidate *int64) ([]definition, error) {
	return s.definitionStore.Definitions(ctx, candidate, s.config.NamespaceID)
}

func mergeTypes(target, source *typecatalog.Catalog) error {
	// Runtime contracts already retain their exact catalog per artifact. The
	// manager seed remains empty until Catalog exposes origin-preserving merge.
	return nil
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.listeners) > 0 {
		return fmt.Errorf("runtime host is already started")
	}
	if err := s.Reload(ctx, 0); err != nil {
		return err
	}
	routes := http.NewServeMux()
	routes.HandleFunc("/_studio/reload", s.reloadHTTP)
	routes.HandleFunc("/_studio/status", s.statusHTTP)
	routes.HandleFunc("/_studio/cache/invalidate", s.cacheInvalidateHTTP)
	routes.Handle("/", browserHTTPCORS(s.config.HTTP.CORS, s.authenticated(s.manager)))
	httpServer := hardenedHTTPServer(s.config.HTTP.Address, routes)
	protocol, err := mcpserver.New(mcpserver.Config{Source: s.manager, Implementation: schema.Implementation{Name: "datly-studio-dynamic", Version: "1"}, Transport: mcpserver.TransportConfig{Kind: mcpserver.TransportStreamable, Address: s.config.MCP.Address, CORS: s.config.MCP.CORS}})
	if err != nil {
		return err
	}
	mcpHTTP, err := protocol.HTTP()
	if err != nil {
		return err
	}
	if s.config.ToolCallAudit != nil {
		wrapped := s.config.ToolCallAudit.Wrap(mcpHTTP.Handler)
		mcpHTTP.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped.ServeHTTP(w, r.WithContext(audit.WithNamespace(r.Context(), s.config.NamespaceID)))
		})
	}
	mcpHTTP.Handler = browserMCPCORS(s.config.MCP.CORS, s.oauthDiscovery(mcpHTTP.Handler))
	if s.config.ForgeProvider != nil {
		providerHandler, providerErr := s.forgeProviderHTTP(ctx)
		if providerErr != nil {
			return providerErr
		}
		mux := http.NewServeMux()
		mux.Handle("/forge/", http.StripPrefix("/forge", providerHandler))
		mux.Handle("/", mcpHTTP.Handler)
		mcpHTTP.Handler = mux
	}
	mcpHTTP.ReadHeaderTimeout = 5 * time.Second
	mcpHTTP.ReadTimeout = 30 * time.Second
	mcpHTTP.WriteTimeout = 60 * time.Second
	mcpHTTP.IdleTimeout = 2 * time.Minute
	servers := []*http.Server{httpServer, mcpHTTP}
	listeners, err := listenHTTPServers(servers)
	if err != nil {
		return err
	}
	s.servers = servers
	s.listeners = listeners
	for index, server := range servers {
		listener := listeners[index]
		go func(server *http.Server, listener net.Listener) { _ = server.Serve(listener) }(server, listener)
	}
	if s.releaseSyncController != nil {
		interval := s.config.ReleaseSyncInterval
		if interval <= 0 {
			interval = 2 * time.Second
		}
		workerCtx, cancel := context.WithCancel(context.Background())
		s.releaseSyncCancel = cancel
		s.releaseSyncDone = make(chan struct{})
		go func() {
			defer close(s.releaseSyncDone)
			_ = s.releaseSyncController.RunSync(workerCtx, interval, func(_ bool, err error) {
				if err != nil && workerCtx.Err() == nil {
					slog.Warn("release background reconciliation failed")
				}
			})
		}()
	}
	return nil
}

// Bind every transport before serving any of them. A failed MCP bind must not
// leave an undisclosed HTTP listener accepting requests after Start returns.
func listenHTTPServers(servers []*http.Server) ([]net.Listener, error) {
	listeners := make([]net.Listener, 0, len(servers))
	for _, server := range servers {
		listener, err := net.Listen("tcp", server.Addr)
		if err != nil {
			for _, opened := range listeners {
				_ = opened.Close()
			}
			return nil, err
		}
		listeners = append(listeners, listener)
	}
	return listeners, nil
}

func hardenedHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr: address, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 60 * time.Second, IdleTimeout: 2 * time.Minute,
	}
}

func (s *Service) reloadHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", http.MethodPost)
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !validAdminToken(s.config.Admin.Token, request.Header.Get("X-Studio-Runtime-Token")) {
		http.Error(response, "unauthorized", http.StatusUnauthorized)
		return
	}
	var payload struct {
		Generation int64 `json:"generation"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1024)).Decode(&payload); err != nil || payload.Generation < 0 {
		http.Error(response, "generation must be zero or greater", http.StatusBadRequest)
		return
	}
	if err := s.Reload(request.Context(), payload.Generation); err != nil {
		http.Error(response, err.Error(), http.StatusConflict)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	_, _ = response.Write([]byte(`{"status":"active"}`))
}

func (s *Service) statusHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.Header().Set("Allow", http.MethodGet)
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !validAdminToken(s.config.Admin.Token, request.Header.Get("X-Studio-Runtime-Token")) {
		http.Error(response, "unauthorized", http.StatusUnauthorized)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(struct {
		AuthenticationMode string `json:"authenticationMode"`
		Status             string `json:"status"`
		Revision           int64  `json:"revision"`
	}{AuthenticationMode: s.config.Authentication.DefaultMode, Status: "ready", Revision: int64(s.manager.Revision())})
}

func validAdminToken(expected, actual string) bool {
	return expected != "" && subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func (s *Service) authenticated(next http.Handler) http.Handler {
	if s.config.Access != nil {
		return s.resourceCredential(next)
	}
	if len(s.config.Authentication.Components) > 0 {
		return s.componentAuthenticated(next)
	}
	if strings.EqualFold(s.config.Authentication.DefaultMode, "public") {
		return next
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		parts := strings.Fields(request.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || s.verifier == nil {
			http.Error(response, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := s.verifier.VerifyClaims(request.Context(), parts[1])
		if err != nil || !matchesDefaultIdentity(claims, s.config.Authentication) {
			http.Error(response, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(request.Context(), verifiedClaimsKey{}, claims)
		identity := audit.Identity{Subject: claims.Subject, NamespaceID: s.config.NamespaceID}
		if claims.UserID != 0 {
			identity.UserID = strconv.Itoa(claims.UserID)
		}
		ctx = audit.WithIdentity(ctx, identity)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

func (s *Service) authorizeRun(ctx context.Context, reportID string) error {
	if policy, ok := s.config.Authentication.Components[reportID]; ok {
		if policy.Public {
			return nil
		}
		identities, _ := ctx.Value(providerIdentitiesKey{}).(map[string]*jwt.Claims)
		claims := identities[policy.Provider]
		if claims == nil {
			return &xresponse.Error{Code: http.StatusUnauthorized, Cause: errors.New("component runtime identity provider is required")}
		}
		if !hasScopes(claims, policy.Scopes) {
			return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("component runtime scopes are required")}
		}
		return nil
	}
	if strings.EqualFold(s.config.Authentication.DefaultMode, "public") {
		return nil
	}
	claims, _ := ctx.Value(verifiedClaimsKey{}).(*jwt.Claims)
	if claims == nil || strings.TrimSpace(claims.Subject) == "" {
		return &xresponse.Error{Code: http.StatusUnauthorized, Cause: errors.New("verified JWT subject is required")}
	}
	allowed, err := s.runAccessStore.Allowed(ctx, reportID, claims.Subject)
	if err != nil {
		return err
	}
	if !allowed {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("reader execution permission is required")}
	}
	return nil
}

func (s *Service) Close(ctx context.Context) error {
	if s.releaseSyncCancel != nil {
		s.releaseSyncCancel()
		select {
		case <-s.releaseSyncDone:
		case <-time.After(time.Second):
			slog.Warn("release background reconciliation did not stop before shutdown")
		}
	}
	var result error
	errorsByServer := make(chan error, len(s.servers)+1)
	var wait sync.WaitGroup
	for _, server := range s.servers {
		server.SetKeepAlivesEnabled(false)
		wait.Add(1)
		go func(server *http.Server) {
			defer wait.Done()
			if err := server.Shutdown(ctx); err != nil {
				// A streamable MCP or pinned Datly request can outlive the graceful
				// deadline. Close the listener and remaining connections so shutdown
				// is bounded; report only a failure to perform that forced close.
				if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
					errorsByServer <- fmt.Errorf("shutdown %s: %w; force close: %v", server.Addr, err, closeErr)
					return
				}
				errorsByServer <- nil
				return
			}
			errorsByServer <- nil
		}(server)
	}
	wait.Add(1)
	go func() {
		defer wait.Done()
		if err := s.manager.Shutdown(ctx); err != nil {
			errorsByServer <- fmt.Errorf("shutdown runtime manager: %w", err)
			return
		}
		errorsByServer <- nil
	}()
	wait.Wait()
	close(errorsByServer)
	for err := range errorsByServer {
		result = errors.Join(result, err)
	}
	if s.connectorStore != nil {
		result = errors.Join(result, s.connectorStore.Close(ctx))
	}
	if s.definitionStore != nil {
		result = errors.Join(result, s.definitionStore.Close(ctx))
	}
	if s.runAccessStore != nil {
		result = errors.Join(result, s.runAccessStore.Close(ctx))
	}
	if s.compiledArtifacts != nil {
		result = errors.Join(result, s.compiledArtifacts.Close(ctx))
	}
	if s.namespacePolicyReader != nil {
		result = errors.Join(result, s.namespacePolicyReader.Close(ctx))
	}
	if s.resourceAccess != nil {
		if inherited, ok := s.resourceAccess.Store.(*namespacepolicy.Store); ok {
			if closer, ok := inherited.Store.(interface{ Close(context.Context) error }); ok {
				result = errors.Join(result, closer.Close(ctx))
			}
		}
	}
	result = errors.Join(result, s.closePrimitiveMetadata(ctx))
	result = errors.Join(result, s.studio.Close())
	return result
}

func (s *Service) Addresses() (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.listeners) < 2 {
		return "", ""
	}
	return s.listeners[0].Addr().String(), s.listeners[1].Addr().String()
}

func normalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlite3":
		return "sqlite"
	case "viant/bigquery":
		return "bigquery"
	case "pg", "pgx":
		return "postgres"
	default:
		return strings.TrimSpace(driver)
	}
}
