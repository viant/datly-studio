package access

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/viant/datly-studio/sdk"
)

const OperationGet = "access.get"
const OperationReplace = "access.replace"
const OperationContext = "access.context"

// Transport adds generic ACL management to any existing Studio transport.
// Service.Provider must validate the request credential independently.
type Transport struct {
	Next    sdk.Transport
	Service *Service
}

func (t *Transport) Invoke(ctx context.Context, operation string, input, output any) error {
	if operation != OperationGet && operation != OperationReplace && operation != OperationContext {
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
	var result Document
	if operation == OperationContext {
		out, ok := output.(*EditorContext)
		if !ok {
			return &sdk.Error{Code: sdk.ErrorInternal, Message: "Invalid access output"}
		}
		var resource Resource
		if err = decode(&resource); err != nil {
			return err
		}
		value, contextErr := t.Service.EditorContext(ctx, resource)
		if contextErr != nil {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Resource access is not permitted", Cause: contextErr}
		}
		*out = value
		return nil
	}
	out, ok := output.(*Document)
	if !ok {
		return &sdk.Error{Code: sdk.ErrorInternal, Message: "Invalid access output"}
	}
	if operation == OperationGet {
		var resource Resource
		if err = decode(&resource); err != nil {
			return err
		}
		result, err = t.Service.Get(ctx, resource)
	} else {
		var doc Document
		if err = decode(&doc); err != nil {
			return err
		}
		result, err = t.Service.Replace(ctx, doc)
	}
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "Access policy changed. Reload before saving."}
		}
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Resource access is not permitted", Cause: err}
	}
	*out = result
	return nil
}
