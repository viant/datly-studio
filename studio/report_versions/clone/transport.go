package clone

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

// Transport adds native draft cloning to an SDK host. The supplied invoker must
// expose the same linked components and shared Studio/authz connector identity.
type Transport struct {
	Next    sdk.Transport
	Invoker exec.ComponentInvoker
}

func (t *Transport) Invoke(ctx context.Context, operation string, input, output any) error {
	if operation != sdk.OperationVersionClone {
		if t.Next == nil {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "Unknown SDK operation"}
		}
		return t.Next.Invoke(ctx, operation, input, output)
	}
	if t.Invoker == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "Draft cloning is not configured"}
	}
	namespace, selected := sdk.NamespaceSelectionFromContext(ctx)
	credential, verified := sdk.VerifiedCredentialFromContext(ctx)
	if !selected || !verified || strings.TrimSpace(credential.Bearer) == "" {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Verified namespace and credential are required"}
	}
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://studio.internal/v1/studio/sdk/versions.clone", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(strings.TrimPrefix(credential.Bearer, "Bearer ")))
	request.Header.Set("X-Studio-Namespace", namespace)
	scope, err := requestprovider.New(request)
	if err != nil {
		return err
	}
	defer scope.Close()
	result, err := t.Invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[Component]().PkgPath(), Name: "clone"}, Route: spec.RouteRef{Method: "POST", Path: "/v1/studio/sdk/versions.clone"}}, Providers: scope.Providers()})
	if err != nil {
		return err
	}
	value, ok := result.(*Output)
	target, valid := output.(*sdk.ReportVersion)
	if !ok || value == nil || value.Response == nil || !valid {
		return &sdk.Error{Code: sdk.ErrorInternal, Message: "Invalid native clone response"}
	}
	*target = *value.Response
	return nil
}
