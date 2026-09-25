package host

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/runtime/accesscontext"
	studiors "github.com/viant/datly-studio/runtime/resources"
	"github.com/viant/datly-studio/sdk/access"
	studiohost "github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/application"
	"github.com/viant/datly/authoring/readerbuilder"
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
	skillformat "github.com/viant/mcp-protocol/extension/skills"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/scy/auth/jwt/verifier"
	xresponse "github.com/viant/xdatly/response"
)

type verifiedClaimsKey struct{}

type Service struct {
	resourceAccess    *access.Service
	config            Config
	studio            *sql.DB
	connectorStore    *activeConnectorStore
	definitionStore   *publishedDefinitionStore
	runAccessStore    *runAccessStore
	manager           *application.Manager
	servers           []*http.Server
	listeners         []net.Listener
	verifier          *verifier.Service
	providerVerifiers map[string]*verifier.Service
	secrets           connectorsecret.Resolver
	mu                sync.Mutex
}

func New(ctx context.Context, config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
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
	predicates, err := (studiohost.Config{PredicatePackages: config.PredicatePackages}).PredicateCatalog()
	if err != nil {
		studio.Close()
		return nil, err
	}
	types, err := predicates.RuntimeTypes()
	if err != nil {
		studio.Close()
		return nil, err
	}
	// Published DQL names the access-context output through its import path.
	if err = accesscontext.RegisterTypes(types); err != nil {
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
	result := &Service{config: config, studio: studio, manager: manager, connectorStore: &activeConnectorStore{db: studio}, definitionStore: definitionStore, runAccessStore: runAccessStore}
	if err = result.initResourceAccess(); err != nil {
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
	types, err := seed.Clone()
	if err != nil {
		return nil, err
	}
	versions := make([]studiors.Version, 0, len(definitions))
	for _, definition := range definitions {
		versions = append(versions, studiors.Version{ReportID: definition.reportID, VersionNo: definition.versionNo})
	}
	loadedResources, err := studiors.Load(ctx, s.studio, versions)
	if err != nil {
		return nil, err
	}
	resources := loadedResources.Store
	registrations := make([]*registry.RegisteredComponent, 0, len(definitions))
	reportByComponent := map[spec.Key]string{}
	versionByReport := map[string]int{}
	for _, definition := range definitions {
		versionByReport[definition.reportID] = definition.versionNo
	}
	var opened []*sql.DB
	defer func() {
		if err != nil {
			for _, db := range opened {
				_ = db.Close()
			}
		}
	}()
	active, activeErr := s.activeConnectors(ctx)
	if activeErr != nil {
		return nil, activeErr
	}
	for _, definition := range definitions {
		sources, openErr := openRuntimeSources(ctx, definition, active, s.secrets)
		if openErr != nil {
			err = openErr
			return nil, err
		}
		opened = append(opened, sources.opened...)
		contractResources := loadedResources.ByVersion[studiors.Version{ReportID: definition.reportID, VersionNo: definition.versionNo}]
		contract, compileErr := transcribe.NewCompiler().RuntimeContracts(ctx, s.config.RootDir, &transcribe.Source{Types: types, Scope: definition.scope, Name: definition.name, Text: definition.dql, Connector: definition.connector, Resources: contractResources, ColumnRefiner: column.New(sources.connections)})
		if compileErr != nil {
			err = fmt.Errorf("compile dynamic report %s: %w", definition.reportID, compileErr)
			return nil, err
		}
		if contractErr := requireReaderOnlyContract(contract.Component); contractErr != nil {
			err = fmt.Errorf("compile dynamic report %s: %w", definition.reportID, contractErr)
			return nil, err
		}
		if mergeErr := mergeTypes(types, contract.Types); mergeErr != nil {
			err = mergeErr
			return nil, err
		}
		compilation, buildErr := report.NewProjectCompiler(report.ProjectConfig{Types: contract.Types}).CompileArtifacts([]bootstrap.ArtifactInput{{Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources, CodecFactory: accesscontext.Codecs()}})
		if buildErr != nil {
			err = buildErr
			return nil, err
		}
		compiledRegistrations, registerErr := compilation.RuntimeComponents(ctx, report.RuntimeConfigureFunc(func(_ context.Context, artifact *report.ComponentArtifact) (report.RuntimeCapabilities, error) {
			if artifact.IsReport() {
				return report.RuntimeCapabilities{}, nil
			}
			reader, readerErr := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: sources.sql})
			if readerErr != nil {
				return report.RuntimeCapabilities{}, readerErr
			}
			return report.RuntimeCapabilities{Reader: reader}, nil
		}))
		if registerErr != nil {
			err = registerErr
			return nil, err
		}
		registrations = append(registrations, compiledRegistrations...)
		for _, registered := range compiledRegistrations {
			reportByComponent[registered.Component.Key] = definition.reportID
		}
	}
	if err = validateSkillToolReferences(ctx, s.studio, versions, registrations); err != nil {
		return nil, err
	}
	// Published components bind their access context as an ordinary Datly
	// component dependency; the context component is server-owned and
	// registered in the same generation at that component's concrete route.
	accessContexts, bindsAccessContext, contextErr := s.accessContexts(registrations, reportByComponent, versionByReport)
	if contextErr != nil {
		err = contextErr
		return nil, err
	}
	registrations = append(registrations, accessContexts...)
	authorizeTarget := func(ctx context.Context, target dexec.ComponentTarget) error {
		reportID := reportByComponent[target.Component]
		if reportID == "" {
			return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("published component authorization is unavailable")}
		}
		if s.config.Access != nil {
			return s.authorizeComponentExecution(ctx, access.Resource{Kind: "component", ID: reportID, Version: strconv.Itoa(versionByReport[reportID]), Tenant: s.config.Access.Tenant}, bindsAccessContext[reportID])
		}
		return s.authorizeRun(ctx, reportID)
	}
	authorizeResource := func(ctx context.Context, uri string) error {
		if s.config.Access != nil {
			return s.authorizeBoundResource(ctx, uri)
		}
		reportID := reportForResourceURI(loadedResources.ResourceReports, uri)
		if reportID == "" {
			// Component-backed resources are authorized by AuthorizeTool when invoked.
			return nil
		}
		return s.authorizeRun(ctx, reportID)
	}
	httpConfig := gateway.Config{Warmup: &gateway.WarmupConfig{Timeout: 30 * time.Second, Authorize: func(context.Context, *http.Request, dexec.ComponentTarget) error {
		return errors.New("dynamic warmup administration is available through the Studio SDK")
	}, Completed: func(gateway.WarmupResult, error) {}}, Authorize: func(ctx context.Context, _ *http.Request, target dexec.ComponentTarget) error {
		return authorizeTarget(ctx, target)
	}}
	return &application.Build{
		Components: registrations,
		Types:      types,
		Resources:  resources,
		HTTP:       httpConfig,
		MCP: mcp.Config{Folders: loadedResources.Folders, AuthorizeTool: authorizeTarget, AuthorizeResource: authorizeResource,
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
			var result error
			for _, db := range opened {
				result = errors.Join(result, db.Close())
			}
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
	known := map[string]bool{mcp.SkillListTool: true, mcp.SkillGetTool: true}
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
	opened      []*sql.DB
}

func (s *runtimeSources) Close() {
	if s == nil {
		return
	}
	for _, db := range s.opened {
		_ = db.Close()
	}
}

func (s *Service) activeConnectors(ctx context.Context) (map[string]connectorDefinition, error) {
	return s.connectorStore.Read(ctx)
}

func openRuntimeSources(ctx context.Context, definition definition, active map[string]connectorDefinition, resolver connectorsecret.Resolver) (*runtimeSources, error) {
	available := make([]string, 0, len(active))
	for name := range active {
		available = append(available, name)
	}
	inspection := readerbuilder.New(readerbuilder.Config{Scope: definition.scope, Name: definition.name, AvailableConnectors: available}).Apply(ctx, readerbuilder.Request{DQL: definition.dql, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	required := map[string]bool{definition.connector: true}
	if inspection.Structure != nil {
		for _, function := range inspection.Structure.Functions {
			if strings.EqualFold(function.Name, "use_connector") && len(function.Args) > 1 {
				required[strings.Trim(strings.TrimSpace(function.Args[1]), "'\"")] = true
			}
		}
	}
	result := &runtimeSources{connections: column.Connections{}}
	success := false
	defer func() {
		if !success {
			result.Close()
		}
	}()
	for name := range required {
		item, ok := active[name]
		if !ok {
			return nil, fmt.Errorf("dynamic report %s requires active Studio connector %q", definition.reportID, name)
		}
		resolvedDSN, resolveErr := connectorsecret.Resolve(ctx, resolver, item.dsn, item.secretRef)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve Studio connector %q secret: %w", name, resolveErr)
		}
		if strings.TrimSpace(resolvedDSN) == "" {
			return nil, fmt.Errorf("Studio connector %q has no DSN", name)
		}
		db, err := sql.Open(normalizeDriver(item.driver), resolvedDSN)
		if err != nil {
			return nil, fmt.Errorf("open Studio connector %q: %w", name, err)
		}
		if err = connectorinit.Configure(ctx, db, item.driver, item.options); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("configure Studio connector %q: %w", name, err)
		}
		if err = db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("ping Studio connector %q: %w", name, err)
		}
		result.opened = append(result.opened, db)
		result.connections[name] = db
		if name == definition.connector {
			result.sql = &dsql.SQLComponent{DB: db}
		}
	}
	if result.sql == nil {
		return nil, fmt.Errorf("default Studio connector %q is unavailable", definition.connector)
	}
	for name, db := range result.connections {
		if err := result.sql.RegisterConnector(name, db); err != nil {
			return nil, err
		}
	}
	success = true
	return result, nil
}

func (s *Service) definitions(ctx context.Context, candidate *int64) ([]definition, error) {
	return s.definitionStore.Definitions(ctx, candidate)
}

func mergeTypes(target, source *typecatalog.Catalog) error {
	// Runtime contracts already retain their exact catalog per artifact. The
	// manager seed remains empty until Catalog exposes origin-preserving merge.
	return nil
}

func (s *Service) Start(ctx context.Context) error {
	if err := s.Reload(ctx, 0); err != nil {
		return err
	}
	routes := http.NewServeMux()
	routes.HandleFunc("/_studio/reload", s.reloadHTTP)
	routes.HandleFunc("/_studio/status", s.statusHTTP)
	routes.Handle("/", s.authenticated(s.manager))
	httpServer := hardenedHTTPServer(s.config.HTTP.Address, routes)
	protocol, err := mcpserver.New(mcpserver.Config{Source: s.manager, Implementation: schema.Implementation{Name: "datly-studio-dynamic", Version: "1"}, Transport: mcpserver.TransportConfig{Kind: mcpserver.TransportStreamable, Address: s.config.MCP.Address}})
	if err != nil {
		return err
	}
	mcpHTTP, err := protocol.HTTP()
	if err != nil {
		return err
	}
	mcpHTTP.Handler = s.oauthDiscovery(mcpHTTP.Handler)
	mcpHTTP.ReadHeaderTimeout = 5 * time.Second
	mcpHTTP.ReadTimeout = 30 * time.Second
	mcpHTTP.WriteTimeout = 60 * time.Second
	mcpHTTP.IdleTimeout = 2 * time.Minute
	for _, server := range []*http.Server{httpServer, mcpHTTP} {
		listener, listenErr := net.Listen("tcp", server.Addr)
		if listenErr != nil {
			return listenErr
		}
		s.servers = append(s.servers, server)
		s.listeners = append(s.listeners, listener)
		go func(server *http.Server, listener net.Listener) { _ = server.Serve(listener) }(server, listener)
	}
	return nil
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
		if err != nil {
			http.Error(response, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(request.Context(), verifiedClaimsKey{}, claims)
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
