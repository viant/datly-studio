package access

import (
	"context"
	"github.com/viant/authz"
	sharedapi "github.com/viant/authz/component/api"
	store "github.com/viant/authz/component/store/sql"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	"net/url"
	"reflect"
	"strings"
	"sync"
)

var namedPolicyChanged struct {
	sync.RWMutex
	notify func(context.Context, authz.Resource) error
}
var namedPolicyAccountScope struct {
	sync.RWMutex
	resolve func(context.Context) (string, error)
}

// ConfigureNamedPolicyAccountScope binds native policy IDs to the same
// verified account namespace as release channels. Nil preserves legacy mode.
func ConfigureNamedPolicyAccountScope(resolve func(context.Context) (string, error)) {
	namedPolicyAccountScope.Lock()
	namedPolicyAccountScope.resolve = resolve
	namedPolicyAccountScope.Unlock()
}

// ConfigureNamedPolicyChanged installs one producer-owned post-commit hook.
// A failed projection is returned after the named policy has committed; the
// release controller withdraws stale cohorts and retries from durable state.
func ConfigureNamedPolicyChanged(notify func(context.Context, authz.Resource) error) {
	namedPolicyChanged.Lock()
	namedPolicyChanged.notify = notify
	namedPolicyChanged.Unlock()
}

type NamedPolicyInput struct {
	NamespaceId      *string               `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Auth             *studioauth.Output    `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" json:"-"`
	Jwt              *jwt.Claims           `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim" json:"-"`
	Operation        string                `parameter:"Operation,kind=body,in=operation,required=true" json:"operation"`
	ID               string                `parameter:"ID,kind=body,in=id" json:"id,omitempty"`
	Reference        authz.NamedReference  `parameter:"Reference,kind=body,in=reference" json:"reference,omitempty"`
	Definition       authz.NamedDefinition `parameter:"Definition,kind=body,in=definition" json:"definition,omitempty"`
	Resource         authz.Resource        `parameter:"Resource,kind=body,in=resource" json:"resource,omitempty"`
	ExpectedRevision int64                 `parameter:"ExpectedRevision,kind=body,in=expectedRevision" json:"expectedRevision,omitempty"`
	Actions          []string              `parameter:"Actions,kind=body,in=actions" json:"actions,omitempty"`
}
type NamedPolicyComponent struct {
	Contract xdatly.Component[NamedPolicyInput, sharedapi.NamedOutput] `component:"named_policies,path=/v1/authz/sdk/namedPolicies.manage,method=POST,connector=studio,handler=NewNamedPolicies" mcp:"[{\"kind\":\"tool\",\"name\":\"authz.sdk.namedPolicies.manage\"}]" caseFormat:"lc"`
}

var NamedPolicyDatly = new(NamedPolicyComponent)
var namedPolicyReachability = []reflect.Type{reflect.TypeFor[NamedPolicyComponent](), reflect.TypeFor[NamedPolicyInput]()}

func (NamedPolicyComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewNamedPolicies" {
		return custom.Factory(NewNamedPolicies)
	}
	return nil
}

type namedPoliciesHandler struct{}

func NewNamedPolicies() xhandler.Contract[NamedPolicyInput, sharedapi.NamedOutput] {
	return &namedPoliciesHandler{}
}
func (*namedPoliciesHandler) Exec(ctx context.Context, session xhandler.Session, in *NamedPolicyInput, out *sharedapi.NamedOutput) error {
	if in == nil || out == nil {
		return publicError(400, "request required")
	}
	service, requestCtx, closeStore, err := setup(ctx, session, in.Jwt)
	if err != nil {
		return err
	}
	defer closeStore()
	if in.Auth == nil || in.Auth.Auth == nil || in.Jwt == nil || in.Auth.Auth.Subject != in.Jwt.Subject {
		return publicError(403, "Verified Studio identity is required")
	}
	namedPolicyAccountScope.RLock()
	resolveScope := namedPolicyAccountScope.resolve
	namedPolicyAccountScope.RUnlock()
	id := in.ID
	accountPrefix := ""
	if resolveScope != nil {
		accountID, resolveErr := resolveScope(requestCtx)
		if resolveErr != nil || accountID == "" || strings.TrimSpace(accountID) != accountID {
			return publicError(403, "Verified account scope is required")
		}
		accountPrefix = "account/" + url.PathEscape(accountID) + "/"
		if in.Operation == "create" {
			if strings.HasPrefix(id, "account/") && !strings.HasPrefix(id, accountPrefix) {
				return publicError(403, "Named policy account is unavailable")
			}
			if !strings.HasPrefix(id, accountPrefix) {
				id = accountPrefix + id
			}
		}
	}
	native, ok := service.Store.(*store.Store)
	if !ok {
		return publicError(503, "Policy inventory is unavailable")
	}
	if in.Operation == "assign" {
		if err = checkResourceNamespace(requestCtx, service, in.NamespaceId, in.Resource, in.Auth, in.Jwt); err != nil {
			return err
		}
	}
	admin := creationAdministration(requestCtx, service)
	if in.Operation == "create" && in.NamespaceId != nil {
		facts, resolveErr := service.Provider.Resolve(requestCtx)
		if resolveErr != nil {
			return publicError(403, "Verified policy identity is required")
		}
		namespace := authz.Resource{Kind: "namespace", ID: *in.NamespaceId, Version: "1", Tenant: facts.Tenant}
		if err = checkNamespacePolicy(requestCtx, service, in.NamespaceId, namespace); err != nil {
			return err
		}
		expected := authz.Resource{Kind: "policy", ID: id, Version: "logical", Tenant: facts.Tenant}
		fallback := admin.EditorCheck
		admin.EditorCheck = func(ctx context.Context, current authz.Facts, resource authz.Resource) (bool, error) {
			if resource == expected {
				return canManageNamespacePolicy(ctx, service, current, namespace)
			}
			return fallback(ctx, current, resource)
		}
	}
	services := &sharedapi.Services{Authorization: service, Policies: admin, Named: &authz.NamedAdministration{Policies: admin, Inventory: native}}
	bound := authoritySession{Session: session, binder: authorityBinder{Binder: session.Binder(), services: services}}
	if err := sharedapi.NewNamed().Exec(requestCtx, bound, &sharedapi.NamedInput{JWT: in.Jwt, Operation: in.Operation, ID: id, Reference: in.Reference, Definition: in.Definition, Resource: in.Resource, ExpectedRevision: in.ExpectedRevision, Actions: in.Actions}, out); err != nil {
		return err
	}
	if in.Operation == "update" || in.Operation == "archive" {
		namedPolicyChanged.RLock()
		notify := namedPolicyChanged.notify
		namedPolicyChanged.RUnlock()
		if notify != nil {
			if err := notify(requestCtx, in.Reference.Resource); err != nil {
				return publicError(503, "named policy committed; release projection unavailable")
			}
		}
	}
	return nil
}
