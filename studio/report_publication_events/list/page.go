package list

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	xresponse "github.com/viant/xdatly/response"
)

func (input *PublicationEventsListInput) Init(context.Context) error {
	if input == nil || strings.TrimSpace(input.ReportId) == "" {
		return &xresponse.Error{Code: 400, Cause: errors.New("reportId is required")}
	}
	if input.Input.Offset < 0 {
		return &xresponse.Error{Code: 400, Cause: errors.New("offset must be non-negative")}
	}
	input.Input.Operation = strings.TrimSpace(input.Input.Operation)
	switch input.Input.Operation {
	case "", "publish", "rollback", "unpublish":
	default:
		return &xresponse.Error{Code: 400, Cause: errors.New("operation must be publish, rollback, or unpublish")}
	}
	input.Input.Status = strings.TrimSpace(input.Input.Status)
	switch input.Input.Status {
	case "", "succeeded", "failed":
	default:
		return &xresponse.Error{Code: 400, Cause: errors.New("status must be succeeded or failed")}
	}
	if input.Input.Limit <= 0 {
		input.Input.Limit = 20
	}
	if input.Input.Limit > 100 {
		input.Input.Limit = 100
	}
	return nil
}

func (output *PublicationEventsListOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*PublicationEventsListInput]()).(*PublicationEventsListInput)
	if !ok || input == nil {
		return fmt.Errorf("publication event list bound input is unavailable")
	}
	output.PageLimit = input.Input.Limit
	output.PageOffset = input.Input.Offset
	if output.Items == nil {
		output.Items = []*PublicationEvent{}
	}
	for _, item := range output.Items {
		if item == nil || item.ReportId != input.ReportId || item.EventId == "" {
			return fmt.Errorf("publication event list returned a mismatched row")
		}
		if item.Reason != nil && *item.Reason == "" {
			item.Reason = nil
		}
		if item.FailureCode != nil && *item.FailureCode == "" {
			item.FailureCode = nil
		}
		if item.FailureMessage != nil && *item.FailureMessage == "" {
			item.FailureMessage = nil
		}
	}
	return nil
}
