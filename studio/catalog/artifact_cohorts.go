package catalog

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

var publishedArtifactSource struct {
	sync.RWMutex
	source    artifactcohort.Caller
	authorize func(context.Context) error
}

// ConfigurePublishedArtifactCohorts is an operator startup hook for the native
// Data Studio MCP listener. A configured source is never selected by an MCP
// caller, and its authorizer must verify the configured principal independently
// of the request body. Omit both arguments to leave the tools unavailable.
func ConfigurePublishedArtifactCohorts(source artifactcohort.Caller, authorize func(context.Context) error) error {
	if (source == nil) != (authorize == nil) {
		return artifactcohort.ErrInvalid
	}
	publishedArtifactSource.Lock()
	publishedArtifactSource.source = source
	publishedArtifactSource.authorize = authorize
	publishedArtifactSource.Unlock()
	return nil
}

type ArtifactDiscoverInput struct {
	Jwt       *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim" json:"-"`
	Auth      *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" json:"-"`
	KnownHash string             `parameter:"KnownHash,kind=body,in=knownHash" json:"knownHash,omitempty"`
}
type ArtifactFetchInput struct {
	Jwt    *jwt.Claims            `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim" json:"-"`
	Auth   *studioauth.Output     `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" json:"-"`
	Tenant string                 `parameter:"Tenant,kind=body,in=tenant,required=true" json:"tenant"`
	Root   artifactcohort.RootRef `parameter:"Root,kind=body,in=root,required=true" json:"root"`
}
type ArtifactDiscoverOutput struct {
	Response artifactcohort.Manifest `parameter:"Response,kind=output,in=body,dataType=artifactcohort.Manifest" json:"-"`
}
type ArtifactFetchOutput struct {
	Response *artifactcohort.Artifact `parameter:"Response,kind=output,in=body,dataType=*artifactcohort.Artifact" json:"-"`
}

func (*ArtifactDiscoverOutput) JSONWireType() reflect.Type {
	return reflect.TypeFor[artifactcohort.Manifest]()
}
func (o *ArtifactDiscoverOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }
func (*ArtifactFetchOutput) JSONWireType() reflect.Type {
	return reflect.TypeFor[artifactcohort.Artifact]()
}
func (o *ArtifactFetchOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }

type ArtifactDiscoverComponent struct {
	Contract xdatly.Component[ArtifactDiscoverInput, ArtifactDiscoverOutput] `component:"artifact_discover,path=/v1/studio/catalog/artifact/discover,method=POST,connector=studio,handler=NewArtifactDiscover" mcp:"[{\"kind\":\"tool\",\"name\":\"artifact/discover\"}]" caseFormat:"lc"`
}
type ArtifactFetchComponent struct {
	Contract xdatly.Component[ArtifactFetchInput, ArtifactFetchOutput] `component:"artifact_fetch,path=/v1/studio/catalog/artifact/fetch,method=POST,connector=studio,handler=NewArtifactFetch" mcp:"[{\"kind\":\"tool\",\"name\":\"artifact/fetch\"}]" caseFormat:"lc"`
}

var ArtifactDiscoverDatlyLinkedType = reflect.TypeFor[ArtifactDiscoverComponent]()
var ArtifactFetchDatlyLinkedType = reflect.TypeFor[ArtifactFetchComponent]()

func (ArtifactDiscoverComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewArtifactDiscover" {
		return custom.Factory(NewArtifactDiscover)
	}
	return nil
}
func (ArtifactFetchComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewArtifactFetch" {
		return custom.Factory(NewArtifactFetch)
	}
	return nil
}
func NewArtifactDiscover() xhandler.Contract[ArtifactDiscoverInput, ArtifactDiscoverOutput] {
	return &handler[ArtifactDiscoverInput, ArtifactDiscoverOutput]{run: runArtifactDiscover}
}
func NewArtifactFetch() xhandler.Contract[ArtifactFetchInput, ArtifactFetchOutput] {
	return &handler[ArtifactFetchInput, ArtifactFetchOutput]{run: runArtifactFetch}
}

func artifactSource(ctx context.Context, claims *jwt.Claims, auth *studioauth.Output) (artifactcohort.Caller, error) {
	if claims == nil || claims.Subject == "" || auth == nil || auth.Auth == nil || auth.Auth.Subject != claims.Subject {
		return nil, fail(403, "verified Studio identity is required")
	}
	publishedArtifactSource.RLock()
	source, authorize := publishedArtifactSource.source, publishedArtifactSource.authorize
	publishedArtifactSource.RUnlock()
	if source == nil || authorize == nil {
		return nil, fail(503, "published artifact catalog is unavailable")
	}
	if err := authorize(ctx); err != nil {
		return nil, fail(403, "configured artifact host identity is required")
	}
	return source, nil
}
func validArtifactHash(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}
func runArtifactDiscover(ctx context.Context, _ xhandler.Session, input *ArtifactDiscoverInput, output *ArtifactDiscoverOutput) error {
	if input == nil || output == nil || input.KnownHash != "" && !validArtifactHash(input.KnownHash) {
		return fail(400, "invalid artifact discovery request")
	}
	source, err := artifactSource(ctx, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	manifest, err := source.Discover(ctx)
	if err != nil || artifactcohort.ValidateManifest(manifest) != nil {
		return fail(503, "published artifact catalog is unavailable")
	}
	if input.KnownHash == manifest.Hash {
		output.Response = artifactcohort.Manifest{Unchanged: true, Hash: manifest.Hash}
		return nil
	}
	output.Response = manifest
	return nil
}
func runArtifactFetch(ctx context.Context, _ xhandler.Session, input *ArtifactFetchInput, output *ArtifactFetchOutput) error {
	if input == nil || output == nil || input.Tenant == "" {
		return fail(400, "invalid artifact fetch request")
	}
	source, err := artifactSource(ctx, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	artifact, err := source.Fetch(ctx, artifactcohort.FetchInput{Tenant: input.Tenant, Root: input.Root})
	if err != nil || artifactcohort.ValidateArtifact(artifact) != nil || artifact.Root != input.Root {
		return fail(403, "published artifact is unavailable")
	}
	output.Response = artifact
	return nil
}
