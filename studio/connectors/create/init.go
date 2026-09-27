package create

import (
	"context"
	"errors"
	"strings"
	"time"

	xresponse "github.com/viant/xdatly/response"
)

// Init turns the flat SDK body into a server-owned insert row. Secrets remain
// input-only and owner/status/revision/timestamps never come from the caller.
func (input *ConnectorCreateInput) Init(context.Context) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return &xresponse.Error{Code: 403, Cause: errors.New("verified Studio owner is required")}
	}
	if input.Connector == nil {
		return &xresponse.Error{Code: 400, Cause: errors.New("connector body is required")}
	}
	row := input.Connector
	row.Name, row.Driver = strings.TrimSpace(row.Name), strings.TrimSpace(row.Driver)
	if row.Name == "" || row.Driver == "" {
		return &xresponse.Error{Code: 400, Cause: errors.New("name and driver are required")}
	}
	if row.OwnerId != "" && row.OwnerId != input.Jwt.Subject {
		return &xresponse.Error{Code: 403, Cause: errors.New("cannot create another principal's connector")}
	}
	if row.DsnTemplate != nil && *row.DsnTemplate == "" {
		row.DsnTemplate = nil
	}
	if row.SecretRef != nil && *row.SecretRef == "" {
		row.SecretRef = nil
	}
	if row.Description != nil && *row.Description == "" {
		row.Description = nil
	}
	if len(row.OptionsJson) == 0 {
		row.OptionsJson = []byte("{}")
	}
	now := time.Now().UTC()
	row.OwnerId, row.Status, row.Etag = input.Jwt.Subject, "draft", 1
	row.CreatedAt, row.UpdatedAt = now, now
	row.Has = &ConnectorRecordHas{Name: true, Driver: true, DsnTemplate: true,
		SecretRef: true, Description: true, OwnerId: true, Status: true,
		OptionsJson: true, Etag: true, CreatedAt: true, UpdatedAt: true}
	input.SetConnector(row)
	return nil
}
