package sqltransport

import (
	"context"
	stdsql "database/sql"
	"errors"
	"strings"

	"github.com/viant/datly-studio/sdk"
)

func (t *Transport) acl(ctx context.Context, operation string, input, output any) error {
	if !t.aclAvailable(ctx) {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "ACL management requires authenticated Studio access"}
	}
	var identity struct {
		ReportID, SubjectType, SubjectID string
		ETag                             int64
	}
	if err := decode(input, &identity); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(identity.ReportID) == "" {
		return invalid(errors.New("reportId is required"))
	}
	switch operation {
	case sdk.OperationACLList:
		return t.listACL(ctx, identity.ReportID, output)
	case sdk.OperationACLUpsert:
		var value sdk.ReportACL
		if err := decode(input, &value); err != nil {
			return invalid(err)
		}
		if err := t.requireACLAdmin(ctx, value.ReportID); err != nil {
			return err
		}
		if err := validACLSubject(value.SubjectType, value.SubjectID); err != nil {
			return invalid(err)
		}
		if err := validACLCapabilities(value); err != nil {
			return invalid(err)
		}
		return t.upsertACL(ctx, value, output)
	case sdk.OperationACLDelete:
		if err := t.requireACLAdmin(ctx, identity.ReportID); err != nil {
			return err
		}
		if err := validACLSubject(identity.SubjectType, identity.SubjectID); err != nil {
			return invalid(err)
		}
		current, found, err := t.aclValue(ctx, identity.ReportID, identity.SubjectType, identity.SubjectID)
		if err != nil {
			return internal(err)
		}
		if !found {
			return &sdk.Error{Code: sdk.ErrorNotFound, Message: "ACL entry was not found"}
		}
		if identity.ETag <= 0 || identity.ETag != current.ETag {
			return aclETagConflict(identity.ETag, current.ETag)
		}
		err = t.writeACL(ctx, "patch", aclWriteRow(*current, true))
		if err != nil {
			current, found, lookupErr := t.aclValue(ctx, identity.ReportID, identity.SubjectType, identity.SubjectID)
			if lookupErr != nil {
				return internal(lookupErr)
			}
			if !found {
				return &sdk.Error{Code: sdk.ErrorNotFound, Message: "ACL entry was not found"}
			}
			if current.ETag != identity.ETag {
				return aclETagConflict(identity.ETag, current.ETag)
			}
			return internal(err)
		}
		return nil
	}
	return invalid(errors.New("unsupported ACL operation"))
}

func (t *Transport) listACL(ctx context.Context, reportID string, output any) error {
	if err := t.requireACLAdmin(ctx, reportID); err != nil {
		return err
	}
	items, err := t.readACLList(ctx, reportID)
	if err != nil {
		return internal(err)
	}
	result := struct {
		Items []*sdk.ReportACL `json:"items"`
	}{Items: items}
	return assign(output, &result)
}

// upsertACL makes the version token part of the mutation predicate. This is
// deliberately not a read-modify-write update: concurrent writers can both
// read a token, but exactly one conditional UPDATE can consume it.
func (t *Transport) upsertACL(ctx context.Context, value sdk.ReportACL, output any) error {
	current, found, err := t.aclValue(ctx, value.ReportID, value.SubjectType, value.SubjectID)
	if err != nil {
		return internal(err)
	}
	if found {
		if value.ETag <= 0 || value.ETag != current.ETag {
			return aclETagConflict(value.ETag, current.ETag)
		}
		err := t.writeACL(ctx, "put", aclWriteRow(value, false))
		if err != nil {
			latest, latestFound, lookupErr := t.aclValue(ctx, value.ReportID, value.SubjectType, value.SubjectID)
			if lookupErr != nil {
				return internal(lookupErr)
			}
			if !latestFound {
				return &sdk.Error{Code: sdk.ErrorNotFound, Message: "ACL entry was not found"}
			}
			if latest.ETag != value.ETag {
				return aclETagConflict(value.ETag, latest.ETag)
			}
			return internal(err)
		}
		value.ETag++
		return assign(output, &value)
	}
	value.ETag = 0
	err = t.writeACL(ctx, "post", aclWriteRow(value, false))
	if err != nil {
		// A competing create may have committed after the initial lookup. In
		// that case report the row's current token instead of leaking a driver
		// specific duplicate-key error.
		latest, latestFound, lookupErr := t.aclValue(ctx, value.ReportID, value.SubjectType, value.SubjectID)
		if lookupErr != nil {
			return internal(lookupErr)
		}
		if latestFound {
			return aclETagConflict(value.ETag, latest.ETag)
		}
		return internal(err)
	}
	value.ETag = 1
	return assign(output, &value)
}

func (t *Transport) aclValue(ctx context.Context, reportID, subjectType, subjectID string) (*sdk.ReportACL, bool, error) {
	value, err := t.readACLOne(ctx, reportID, subjectType, subjectID)
	if errors.Is(err, stdsql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func aclETagConflict(expected, current int64) *sdk.Error {
	return &sdk.Error{Code: sdk.ErrorConflict, Message: "ACL etag does not match", ExpectedETag: expected, CurrentETag: current}
}

func (t *Transport) requireACLAdmin(ctx context.Context, reportID string) error {
	report, err := t.getReportValue(ctx, reportID)
	if err != nil {
		return err
	}
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return nil // trusted in-process administration
	}
	if principal.Subject != report.OwnerID {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "only the report owner can administer access"}
	}
	return nil
}

func validACLSubject(subjectType, subjectID string) error {
	subjectType, subjectID = strings.TrimSpace(subjectType), strings.TrimSpace(subjectID)
	if subjectType != "user" || subjectID == "" || len(subjectID) > 128 {
		return errors.New("subjectType must be user and subjectId must be a verified JWT sub")
	}
	return nil
}

func validACLCapabilities(value sdk.ReportACL) error {
	if (value.CanRun || value.CanEdit || value.CanPublish || value.CanUseDQL) && !value.CanView {
		return errors.New("view capability is required for run, edit, publish, or DQL access")
	}
	if value.CanUseDQL && !value.CanEdit {
		return errors.New("edit capability is required for advanced DQL access")
	}
	return nil
}
