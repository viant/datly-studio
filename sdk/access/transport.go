package access

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/viant/authz"
	"io"

	"github.com/viant/datly-studio/sdk"
	resourcecatalog "github.com/viant/datly-studio/studio/resource_policy/catalog"
)

const OperationList = "access.list"

const OperationGet = "access.get"
const OperationReplace = "access.replace"
const OperationContext = "access.context"

// Transport adds generic ACL management to any existing Studio transport.
// Service.Provider must validate the request credential independently.
type Transport struct {
	Next    sdk.Transport
	Service *authz.Service
	Catalog *Catalog
}

func (t *Transport) Invoke(ctx context.Context, operation string, input, output any) error {
	if operation != OperationGet && operation != OperationReplace && operation != OperationContext && operation != OperationList {
		if t.Next == nil {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "Unknown SDK operation"}
		}
		return t.Next.Invoke(ctx, operation, input, output)
	}
	if t.Service == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "Resource access management is not configured"}
	}
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	decode := func(target any) error {
		d := json.NewDecoder(bytes.NewReader(body))
		d.DisallowUnknownFields()
		if e := d.Decode(target); e != nil {
			return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "Invalid access request"}
		}
		if e := d.Decode(new(any)); e != io.EOF {
			return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "Invalid access request"}
		}
		return nil
	}
	if operation == OperationList {
		var in CatalogInput
		if err = decode(&in); err != nil {
			return err
		}
		out, ok := output.(*CatalogPage)
		if !ok {
			return &sdk.Error{Code: sdk.ErrorInternal, Message: "Invalid catalog output"}
		}
		catalog := t.Catalog
		if catalog == nil {
			catalog = &Catalog{Service: t.Service}
		}
		if id, present := sdk.NamespaceSelectionFromContext(ctx); present {
			copy := *catalog
			prior := copy.ResourceScope
			copy.ResourceScope = func(ctx context.Context, row *resourcecatalog.Entry) bool {
				return row.NamespaceID == id && (prior == nil || prior(ctx, row)) && t.canViewComponent(ctx, row.ComponentID)
			}
			catalog = &copy
		}
		value, listErr := catalog.List(ctx, in)
		if listErr != nil {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Resource catalog is not permitted", Cause: listErr}
		}
		*out = value
		return nil
	}
	var result authz.Document
	if operation == OperationContext {
		out, ok := output.(*authz.EditorContext)
		if !ok {
			return &sdk.Error{Code: sdk.ErrorInternal, Message: "Invalid access output"}
		}
		var resource authz.Resource
		if err = decode(&resource); err != nil {
			return err
		}
		if err = t.checkNamespace(ctx, resource); err != nil {
			return err
		}
		value, contextErr := t.Service.EditorContext(ctx, resource)
		if contextErr != nil {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Resource access is not permitted", Cause: contextErr}
		}
		*out = value
		return nil
	}
	out, ok := output.(*authz.Document)
	if !ok {
		return &sdk.Error{Code: sdk.ErrorInternal, Message: "Invalid access output"}
	}
	if operation == OperationGet {
		var resource authz.Resource
		if err = decode(&resource); err != nil {
			return err
		}
		if err = t.checkNamespace(ctx, resource); err != nil {
			return err
		}
		result, err = t.Service.Get(ctx, resource)
	} else {
		var doc authz.Document
		if err = decode(&doc); err != nil {
			return err
		}
		if err = t.checkNamespace(ctx, doc.Resource); err != nil {
			return err
		}
		result, err = t.Service.Replace(ctx, doc)
	}
	if err != nil {
		if errors.Is(err, authz.ErrConflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "Access policy changed. Reload before saving."}
		}
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Resource access is not permitted", Cause: err}
	}
	*out = result
	return nil
}

func (t *Transport) checkNamespace(ctx context.Context, resource authz.Resource) error {
	id, present := sdk.NamespaceSelectionFromContext(ctx)
	if !present {
		return nil
	}
	var source catalogStore
	if t.Catalog != nil {
		source = t.Catalog.Source
	}
	if source == nil {
		source, _ = t.Service.Store.(catalogStore)
	}
	err := CheckNamespaceResource(ctx, &id, resource, source, t.canViewComponent)
	if err != nil {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Resource access is not permitted", Cause: err}
	}
	return nil
}

func (t *Transport) canViewComponent(ctx context.Context, id string) bool {
	if t.Next == nil || id == "" {
		return false
	}
	var component sdk.Component
	return t.Next.Invoke(ctx, sdk.OperationComponentGet, map[string]any{"id": id}, &component) == nil && component.ID == id
}
