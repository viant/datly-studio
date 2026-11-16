package sqltransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/datly-studio/sdk"
	versiondelete "github.com/viant/datly-studio/studio/report_versions/delete"
	componentdelete "github.com/viant/datly-studio/studio/reports/delete"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	xresponse "github.com/viant/xdatly/response"
)

// deleteNative keeps the SDK on the same owner-authorized native operation as
// HTTP and MCP. The host supplies the existing linked invoker and credential.
func (t *Transport) deleteNative(ctx context.Context, operation string, input, output any) error {
	if t.ComponentInvoker == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "Native deletion is not configured"}
	}
	ns, selected := sdk.NamespaceSelectionFromContext(ctx)
	credential, verified := sdk.VerifiedCredentialFromContext(ctx)
	if !selected || !verified || strings.TrimSpace(credential.Bearer) == "" {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Verified namespace and credential are required"}
	}
	data, err := json.Marshal(input)
	if err != nil {
		return invalid(err)
	}
	path := "/v1/studio/sdk/" + operation
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://studio.internal"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(strings.TrimPrefix(credential.Bearer, "Bearer ")))
	request.Header.Set("X-Studio-Namespace", ns)
	scope, err := requestprovider.New(request)
	if err != nil {
		return err
	}
	defer scope.Close()
	typ := reflect.TypeFor[componentdelete.Component]()
	if operation == sdk.OperationVersionDelete {
		typ = reflect.TypeFor[versiondelete.Component]()
	}
	value, err := t.ComponentInvoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: typ.PkgPath(), Name: "delete"}, Route: spec.RouteRef{Method: http.MethodPost, Path: path}}, Providers: scope.Providers()})
	if err != nil {
		var public *xresponse.Error
		if errors.As(err, &public) {
			code := sdk.ErrorInternal
			switch public.StatusCode() {
			case 400:
				code = sdk.ErrorInvalidArgument
			case 401:
				code = sdk.ErrorUnauthorized
			case 403:
				code = sdk.ErrorForbidden
			case 404:
				code = sdk.ErrorNotFound
			case 409:
				code = sdk.ErrorConflict
			case 503:
				code = sdk.ErrorUnavailable
			}
			message := "Native deletion failed"
			if payload, ok := public.Payload.(map[string]string); ok {
				message = payload["message"]
			}
			if payload, ok := public.Payload.(map[string]any); ok {
				if safe, ok := payload["message"].(string); ok {
					message = safe
				}
			}
			return &sdk.Error{Code: code, Message: message, Cause: err}
		}
		return err
	}
	var result *sdk.DeletionResult
	switch v := value.(type) {
	case *componentdelete.Output:
		result = v.Response
	case *versiondelete.Output:
		result = v.Response
	}
	if result == nil {
		return &sdk.Error{Code: sdk.ErrorInternal, Message: "Invalid native deletion response"}
	}
	return assign(output, result)
}
