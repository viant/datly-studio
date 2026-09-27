package create

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/namespacevalidation"
	xresponse "github.com/viant/xdatly/response"
)

// Init turns the flat SDK body into a server-owned insert row. Caller-supplied
// owner, status, etag, and timestamps cannot override verified identity/state.
func (input *NamespaceCreateInput) Init(context.Context) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return &xresponse.Error{Code: 403, Cause: errors.New("verified Studio owner is required")}
	}
	if input.Namespace == nil {
		return &xresponse.Error{Code: 400, Cause: errors.New("namespace body is required")}
	}
	row := input.Namespace
	row.Name = strings.TrimSpace(row.Name)
	row.Title = strings.TrimSpace(row.Title)
	if !namespacevalidation.ValidName(row.Name) || row.Title == "" {
		return &xresponse.Error{Code: 400, Cause: errors.New("namespace name and title are required; name must use lowercase dot-separated segments")}
	}
	if row.OwnerId != "" && row.OwnerId != input.Jwt.Subject {
		return &xresponse.Error{Code: 403, Cause: errors.New("cannot create another principal's namespace")}
	}
	if row.Description != nil && *row.Description == "" {
		row.Description = nil
	}
	now, etag := time.Now().UTC(), int64(1)
	row.OwnerId, row.Status = input.Jwt.Subject, "active"
	row.Etag, row.CreatedAt, row.UpdatedAt, row.DeletedAt = &etag, &now, &now, nil
	row.Has = &NamespaceRecordHas{OwnerId: true, Name: true, Title: true,
		Description: true, Status: true, Etag: true, CreatedAt: true, UpdatedAt: true}
	input.SetNamespace(row)
	return nil
}
