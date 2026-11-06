package host

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly/application"
	"github.com/viant/datly/bootstrap"
	runtimeauth "github.com/viant/datly/runtime/auth"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/scy/auth/jwt/verifier"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type internalJSONInput struct {
	Text string      `parameter:"Text,kind=form,in=text,required"`
	JWT  *jwt.Claims `parameter:"JWT,kind=header,in=Authorization,dataType=string,required,errorCode=401" codec:"JwtClaim" json:"-"`
}
type internalJSONOutput struct {
	Label   string `json:"display_label"`
	ActorID int    `json:"actor_id"`
	Hidden  string `json:"-"`
}
type internalJSONHandler struct{ after func() }

func (h internalJSONHandler) Exec(_ context.Context, _ xhandler.Session, input *internalJSONInput, output *internalJSONOutput) error {
	output.Label = input.Text
	output.ActorID = input.JWT.UserID
	output.Hidden = "private-field"
	if h.after != nil {
		h.after()
	}
	return nil
}

func TestInternalComponentJSONUsesNativeBindingEncodingAndFences(t *testing.T) {
	testDynamicHost(t, false, false, dynamicHostExtension{verify: func(service *Service, _ string) {
		previousConfig, previousAccess := service.config, service.resourceAccess
		defer func() { service.config, service.resourceAccess = previousConfig, previousAccess }()
		facts := &linkedFacts{deadline: time.Now().Add(time.Minute), roles: []string{"reader"}}
		store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "echo", Version: "fixture", Tenant: "team"}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}})
		require.NoError(t, err)
		service.config.Access = &ResourceAccessConfig{Tenant: "team", Provider: facts}
		service.resourceAccess = &authz.Service{Provider: facts, Store: store}
		const secret = "invented-internal-json-key"
		token, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, jwtv5.MapClaims{"sub": "person", "user_id": 7, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(secret))
		require.NoError(t, err)
		ctx := oauth.WithBearer(t.Context(), token)
		var after func()
		internal := true
		build := func(ctx context.Context, _ *typecatalog.Catalog) (*application.Build, error) {
			codecs, err := runtimeauth.New(ctx, &runtimeauth.Config{JWTValidator: &verifier.Config{HMAC: &scy.Resource{URL: "fixture-key", Data: []byte(secret)}}})
			if err != nil {
				return nil, err
			}
			component := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/echo", Name: "Echo"}, Settings: &spec.Settings{CaseFormat: "lc"}, Routes: []*spec.Route{{Method: http.MethodGet, Path: "/echo", Internal: internal}}}
			artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[internalJSONInput](), OutputType: reflect.TypeFor[internalJSONOutput](), CodecFactory: codecs})
			if err != nil {
				return nil, err
			}
			registered, err := artifact.Registration(registry.RegisteredComponent{Handler: custom.New[internalJSONInput, internalJSONOutput](internalJSONHandler{after})})
			if err != nil {
				return nil, err
			}
			return &application.Build{Components: []*registry.RegisteredComponent{registered}}, nil
		}
		service.config.LinkedComponents = []LinkedComponentSource{{ID: "echo", Revision: "fixture", ArtifactFingerprint: identity.ContentFingerprint([]byte("echo-fixture")), Build: build}}
		ref := ComponentReference{Kind: "linked", ID: "echo", Revision: "fixture", Method: http.MethodGet, Route: "/echo"}
		pin, err := service.ResolveComponentBinding(ctx, ref)
		require.NoError(t, err)
		manager, _, err := service.componentRuntime(ctx, ref)
		require.NoError(t, err)
		_, err = executeRuntimeJSON(ctx, service.authenticated(manager), http.MethodGet, "/echo", map[string]any{"text": "not externally exposed"})
		require.ErrorContains(t, err, "404")
		require.NoError(t, manager.Shutdown(ctx))
		raw, err := service.ExecuteComponentJSON(ctx, ref, pin, map[string]any{"text": "native form"})
		require.NoError(t, err)
		require.JSONEq(t, `{"display_label":"native form","actor_id":7}`, string(raw))
		expired, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, jwtv5.MapClaims{"sub": "person", "user_id": 7, "exp": time.Now().Add(-time.Minute).Unix()}).SignedString([]byte(secret))
		require.NoError(t, err)
		for _, bearer := range []string{"", "invalid-token", expired} {
			raw, err = service.ExecuteComponentJSON(oauth.WithBearer(t.Context(), bearer), ref, pin, map[string]any{"text": "denied"})
			require.Error(t, err)
			require.Empty(t, raw)
		}
		facts.roles = nil
		raw, err = service.ExecuteComponentJSON(ctx, ref, pin, map[string]any{"text": "denied"})
		require.Error(t, err)
		require.Empty(t, raw)
		facts.roles = []string{"reader"}
		after = func() {
			service.config.LinkedComponents[0].ArtifactFingerprint = identity.ContentFingerprint([]byte("changed-artifact"))
		}
		raw, err = service.ExecuteComponentJSON(ctx, ref, pin, map[string]any{"text": "withheld"})
		require.Error(t, err)
		require.Empty(t, raw)
		after = nil
		service.config.LinkedComponents[0].ArtifactFingerprint = identity.ContentFingerprint([]byte("echo-fixture"))
		internal = false
		externalPin, err := service.ResolveComponentBinding(ctx, ref)
		require.NoError(t, err)
		raw, err = service.ExecuteComponentJSON(ctx, ref, externalPin, map[string]any{"text": "external form"})
		require.NoError(t, err)
		require.JSONEq(t, `{"display_label":"external form","actor_id":7}`, string(raw))
	}})
}

type internalJSONPostPayload struct {
	Text    string `json:"text"`
	Enabled bool   `json:"enabled"`
}
type internalJSONPostInput struct {
	Payload *internalJSONPostPayload `parameter:"Payload,kind=body,in=payload,required"`
	JWT     *jwt.Claims              `parameter:"JWT,kind=header,in=Authorization,dataType=string,required,errorCode=401" codec:"JwtClaim" json:"-"`
}
type internalJSONResponseOutput struct{ *xresponse.Buffered }
type internalJSONResponseHandler struct{ status, writerStatus int }

func (h internalJSONResponseHandler) Exec(_ context.Context, session xhandler.Session, input *internalJSONPostInput, output *internalJSONResponseOutput) error {
	raw, err := json.Marshal(struct {
		Text    string `json:"text"`
		Enabled bool   `json:"enabled"`
		ActorID int    `json:"actor_id"`
	}{input.Payload.Text, input.Payload.Enabled, input.JWT.UserID})
	if err != nil {
		return err
	}
	if h.writerStatus != 0 {
		session.Response().SetStatusCode(h.writerStatus)
	}
	output.Buffered = xresponse.NewBuffered(xresponse.WithBytes(raw), xresponse.WithStatusCode(h.status), xresponse.WithHeader("Content-Type", "application/json"), xresponse.WithHeader("X-Native-Response", "retained"))
	return nil
}

func TestInternalComponentJSONPreservesTransportResponseAndPOSTBinding(t *testing.T) {
	testDynamicHost(t, false, false, dynamicHostExtension{verify: func(service *Service, _ string) {
		previousConfig, previousAccess := service.config, service.resourceAccess
		defer func() { service.config, service.resourceAccess = previousConfig, previousAccess }()
		facts := &linkedFacts{deadline: time.Now().Add(time.Minute), roles: []string{"reader"}}
		store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "post-response", Version: "fixture", Tenant: "team"}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}})
		require.NoError(t, err)
		service.config.Access = &ResourceAccessConfig{Tenant: "team", Provider: facts}
		service.resourceAccess = &authz.Service{Provider: facts, Store: store}
		const secret = "invented-post-response-key"
		token, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, jwtv5.MapClaims{"sub": "person", "user_id": 9, "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(secret))
		require.NoError(t, err)
		ctx := oauth.WithBearer(t.Context(), token)
		internal := true
		status := http.StatusUnprocessableEntity
		writerStatus := 0
		build := func(ctx context.Context, _ *typecatalog.Catalog) (*application.Build, error) {
			codecs, err := runtimeauth.New(ctx, &runtimeauth.Config{JWTValidator: &verifier.Config{HMAC: &scy.Resource{URL: "fixture-post-key", Data: []byte(secret)}}})
			if err != nil {
				return nil, err
			}
			component := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/response", Name: "Response"}, Routes: []*spec.Route{{Method: http.MethodPost, Path: "/post-response", Internal: internal}}}
			artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[internalJSONPostInput](), OutputType: reflect.TypeFor[internalJSONResponseOutput](), CodecFactory: codecs})
			if err != nil {
				return nil, err
			}
			registered, err := artifact.Registration(registry.RegisteredComponent{Handler: custom.New[internalJSONPostInput, internalJSONResponseOutput](internalJSONResponseHandler{status: status, writerStatus: writerStatus})})
			if err != nil {
				return nil, err
			}
			return &application.Build{Components: []*registry.RegisteredComponent{registered}}, nil
		}
		service.config.LinkedComponents = []LinkedComponentSource{{ID: "post-response", Revision: "fixture", ArtifactFingerprint: identity.ContentFingerprint([]byte("post-response-fixture")), Build: build}}
		ref := ComponentReference{Kind: "linked", ID: "post-response", Revision: "fixture", Method: http.MethodPost, Route: "/post-response"}
		inputs := map[string]any{"payload": map[string]any{"text": "quoted \"body\"", "enabled": false}}
		for _, visibility := range []bool{true, false} {
			internal = visibility
			pin, err := service.ResolveComponentBinding(ctx, ref)
			require.NoError(t, err)
			raw, err := service.ExecuteComponentJSON(ctx, ref, pin, inputs)
			require.ErrorContains(t, err, "422", "a native error response with valid JSON must never become success")
			require.Empty(t, raw)
			status = http.StatusOK
			raw, err = service.ExecuteComponentJSON(ctx, ref, pin, inputs)
			require.NoError(t, err)
			require.JSONEq(t, `{"text":"quoted \"body\"","enabled":false,"actor_id":9}`, string(raw))
			metadata := &exactComponentMetadata{}
			manager, _, err := service.componentRuntimeInspected(ctx, ref, true, metadata)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, ref.Route, strings.NewReader(`{"payload":{"text":"quoted \"body\"","enabled":false}}`)).WithContext(ctx)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			if internal {
				service.internalComponentJSONHandler(manager, metadata, ref).ServeHTTP(response, request)
			} else {
				service.authenticated(manager).ServeHTTP(response, request)
			}
			require.Equal(t, http.StatusOK, response.Code)
			require.Equal(t, "retained", response.Header().Get("X-Native-Response"))
			require.Equal(t, "application/json", response.Header().Get("Content-Type"))
			require.Equal(t, strconv.Itoa(response.Body.Len()), response.Header().Get("Content-Length"))
			require.JSONEq(t, `{"text":"quoted \"body\"","enabled":false,"actor_id":9}`, response.Body.String())
			require.NoError(t, manager.Shutdown(ctx))
			writerStatus = http.StatusConflict
			raw, err = service.ExecuteComponentJSON(ctx, ref, pin, inputs)
			require.ErrorContains(t, err, "409")
			require.Empty(t, raw)
			writerStatus = 0
			raw, err = service.ExecuteComponentJSON(oauth.WithBearer(t.Context(), "invalid-token"), ref, pin, inputs)
			require.Error(t, err)
			require.Empty(t, raw)
			status = http.StatusUnprocessableEntity
		}
	}})
}
