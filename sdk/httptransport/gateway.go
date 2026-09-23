// Package httptransport adapts an HTTP request to the public Studio SDK
// Transport. It contains no SQL, Datly component, or control-service logic.
package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/viant/datly-studio/sdk"
)

const PathPrefix = "/v1/studio/sdk/"

type Mode string

const (
	Development   Mode = "development"
	Authenticated Mode = "authenticated"
)

// Authenticator verifies an authenticated request and returns a context that
// carries the verified principal. JWT/OIDC verification belongs here, outside
// browser configuration and outside the SDK transport.
type Authenticator interface {
	Authenticate(context.Context, *http.Request) (context.Context, error)
}

type AuthenticatorFunc func(context.Context, *http.Request) (context.Context, error)

func (f AuthenticatorFunc) Authenticate(ctx context.Context, request *http.Request) (context.Context, error) {
	return f(ctx, request)
}

type Config struct {
	Mode               Mode
	DevelopmentSubject string
	Authenticator      Authenticator
}

func (c Config) validate() error {
	switch c.Mode {
	case Development:
		if strings.TrimSpace(c.DevelopmentSubject) == "" {
			return errors.New("development SDK gateway requires a development subject")
		}
	case Authenticated:
		if c.Authenticator == nil {
			return errors.New("authenticated SDK gateway requires an authenticator")
		}
	default:
		return fmt.Errorf("unsupported SDK gateway mode %q", c.Mode)
	}
	return nil
}

type Gateway struct {
	Config    Config
	Transport sdk.Transport
}

func (g Gateway) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if err := g.Config.validate(); err != nil {
		writeError(response, http.StatusInternalServerError, &sdk.Error{Code: sdk.ErrorInternal, Message: err.Error()})
		return
	}
	if g.Transport == nil {
		writeError(response, http.StatusServiceUnavailable, &sdk.Error{Code: sdk.ErrorUnavailable, Message: "Studio SDK transport is unavailable"})
		return
	}
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", http.MethodPost)
		writeError(response, http.StatusMethodNotAllowed, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "only POST is supported"})
		return
	}
	operation, ok := strings.CutPrefix(request.URL.Path, PathPrefix)
	if !ok || operation == "" || strings.Contains(operation, "/") {
		writeError(response, http.StatusNotFound, &sdk.Error{Code: sdk.ErrorNotFound, Message: "Studio SDK operation was not found"})
		return
	}
	output, known := outputFor(operation)
	if !known {
		writeError(response, http.StatusNotFound, &sdk.Error{Code: sdk.ErrorNotFound, Message: "Studio SDK operation was not found"})
		return
	}
	ctx, err := g.authenticate(request)
	if err != nil {
		writeError(response, http.StatusUnauthorized, &sdk.Error{Code: sdk.ErrorUnauthorized, Message: "Studio authentication is required", Cause: err})
		return
	}
	limit := int64(1 << 20)
	if operation == sdk.OperationVersionLoadDQL || operation == sdk.OperationVersionLoadArchive {
		limit = 24 << 20
	}
	body, err := io.ReadAll(http.MaxBytesReader(response, request.Body, limit))
	if err != nil {
		writeError(response, http.StatusRequestEntityTooLarge, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "Studio SDK request is too large"})
		return
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}
	if !json.Valid(body) {
		writeError(response, http.StatusBadRequest, &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "Studio SDK input must be JSON"})
		return
	}
	if err = g.Transport.Invoke(ctx, operation, json.RawMessage(body), output); err != nil {
		writeTransportError(response, err)
		return
	}
	if output == nil {
		response.WriteHeader(http.StatusNoContent)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(response).Encode(output); err != nil {
		writeError(response, http.StatusInternalServerError, &sdk.Error{Code: sdk.ErrorInternal, Message: "encode Studio SDK response", Cause: err})
	}
}

func (g Gateway) authenticate(request *http.Request) (context.Context, error) {
	ctx := request.Context()
	switch g.Config.Mode {
	case Development:
		if !isLoopback(request.RemoteAddr) || request.Header.Get("Authorization") != "" ||
			request.Header.Get("X-Studio-Development-Subject") != g.Config.DevelopmentSubject {
			return nil, errors.New("invalid development request")
		}
		return sdk.WithPrincipal(ctx, sdk.Principal{Subject: g.Config.DevelopmentSubject, Development: true}), nil
	case Authenticated:
		if request.Header.Get("X-Studio-Development-Subject") != "" {
			return nil, errors.New("development identity is disabled")
		}
		ctx, err := g.Config.Authenticator.Authenticate(ctx, request)
		if err != nil {
			return nil, err
		}
		if _, ok := sdk.PrincipalFromContext(ctx); !ok {
			return nil, errors.New("authenticated identity is unavailable")
		}
		return ctx, nil
	default:
		return nil, errors.New("invalid SDK gateway mode")
	}
}

func isLoopback(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func outputFor(operation string) (any, bool) {
	switch operation {
	case sdk.OperationConnectorCreate, sdk.OperationConnectorGet, sdk.OperationConnectorUpdate, sdk.OperationConnectorActivate, sdk.OperationConnectorDisable:
		return new(sdk.Connector), true
	case sdk.OperationConnectorList:
		return new(sdk.ConnectorPage), true
	case sdk.OperationConnectorTest:
		return new(sdk.ConnectorTestResult), true
	case sdk.OperationConnectorSchemas:
		return new(sdk.SchemaCatalog), true
	case sdk.OperationVersionLoadDQL, sdk.OperationVersionLoadArchive:
		return new(sdk.DQLLoadResult), true
	case sdk.OperationVersionDownload:
		return new(sdk.ComponentDownload), true
	case sdk.OperationConnectorTables:
		return new(sdk.TableCatalog), true
	case sdk.OperationConnectorTable:
		return new(sdk.TableDetail), true
	case sdk.OperationConnectorTestSQL:
		return new(sdk.SQLTestResult), true
	case sdk.OperationConnectorDelete:
		return nil, true
	case sdk.OperationNamespaceCreate, sdk.OperationNamespaceGet, sdk.OperationNamespaceUpdate:
		return new(sdk.Namespace), true
	case sdk.OperationNamespaceList:
		return new(sdk.NamespacePage), true
	case sdk.OperationAuthorizationPredicateCreate, sdk.OperationAuthorizationPredicateGet, sdk.OperationAuthorizationPredicateUpdate:
		return new(sdk.AuthorizationPredicate), true
	case sdk.OperationAuthorizationPredicateList:
		return new(sdk.AuthorizationPredicatePage), true
	case sdk.OperationAuthorizationPredicateTypes:
		return new(sdk.AuthorizationPredicateTypePage), true
	case sdk.OperationAuthorizationPredicateDelete:
		return nil, true
	case sdk.OperationNamespaceDelete:
		return nil, true
	case sdk.OperationReportCreate, sdk.OperationReportGet, sdk.OperationReportUpdate:
		return new(sdk.Report), true
	case sdk.OperationReportList:
		return new(sdk.ReportPage), true
	case sdk.OperationVersionCreate, sdk.OperationVersionGet:
		return new(sdk.ReportVersion), true
	case sdk.OperationVersionList:
		return new(sdk.VersionPage), true
	case sdk.OperationVersionApply:
		return new(sdk.EditResult), true
	case sdk.OperationVersionValidate:
		return new(sdk.ValidationResult), true
	case sdk.OperationVersionDescriptor:
		return new(sdk.ComponentDescriptor), true
	case sdk.OperationVersionExportDQL:
		return new(sdk.DQLExport), true
	case sdk.OperationVersionInspect:
		return new(sdk.ReaderInspection), true
	case sdk.OperationVersionBuilder:
		return new(sdk.ReaderBuilderResult), true
	case sdk.OperationVersionTestView:
		return new(sdk.ViewTestResult), true
	case sdk.OperationVersionTestRelation:
		return new(sdk.RelationTestResult), true
	case sdk.OperationVersionTestCompose:
		return new(sdk.CubeComposeTestResult), true
	case sdk.OperationVersionWarmup:
		return new(sdk.WarmupResult), true
	case sdk.OperationVersionWarmupGet:
		return new(sdk.WarmupRun), true
	case sdk.OperationVersionWarmupList:
		return new(sdk.WarmupRunPage), true
	case sdk.OperationPreviewExecute:
		return new(sdk.PreviewResult), true
	case sdk.OperationPublicationGet, sdk.OperationPublicationPublish, sdk.OperationPublicationRemove, sdk.OperationPublicationRollback:
		return new(sdk.Publication), true
	case sdk.OperationPublicationEventsList:
		return new(sdk.PublicationEventPage), true
	case sdk.OperationRuntimeStatus:
		return new(sdk.RuntimeStatus), true
	case sdk.OperationResourcesGet, sdk.OperationResourcesUpsertFile, sdk.OperationResourcesDeleteFile,
		sdk.OperationResourcesUpsertFolder, sdk.OperationResourcesDeleteFolder, sdk.OperationResourcesUpsertSkill, sdk.OperationResourcesDeleteSkill:
		return new(sdk.ResourceSnapshot), true
	case sdk.OperationACLList:
		return new(struct {
			Items []*sdk.ReportACL `json:"items"`
		}), true
	case sdk.OperationACLUpsert:
		return new(sdk.ReportACL), true
	case sdk.OperationACLDelete:
		return nil, true
	}
	return nil, false
}

func writeTransportError(response http.ResponseWriter, err error) {
	var sdkError *sdk.Error
	if errors.As(err, &sdkError) {
		writeError(response, statusFor(sdkError.Code), sdkError)
		return
	}
	writeError(response, http.StatusInternalServerError, &sdk.Error{Code: sdk.ErrorInternal, Message: "Studio SDK operation failed", Cause: err})
}

func statusFor(code sdk.ErrorCode) int {
	switch code {
	case sdk.ErrorInvalidArgument:
		return http.StatusBadRequest
	case sdk.ErrorNotFound:
		return http.StatusNotFound
	case sdk.ErrorConflict:
		return http.StatusConflict
	case sdk.ErrorUnauthorized:
		return http.StatusUnauthorized
	case sdk.ErrorForbidden:
		return http.StatusForbidden
	case sdk.ErrorUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func writeError(response http.ResponseWriter, status int, value *sdk.Error) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}
