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
		result, err := t.DB.ExecContext(ctx, `DELETE FROM report_acl WHERE report_id=? AND subject_type=? AND subject_id=? AND etag=?`, identity.ReportID, identity.SubjectType, identity.SubjectID, identity.ETag)
		if err != nil {
			return internal(err)
		}
		if count, _ := result.RowsAffected(); count == 0 {
			current, found, lookupErr := t.aclValue(ctx, identity.ReportID, identity.SubjectType, identity.SubjectID)
			if lookupErr != nil {
				return internal(lookupErr)
			}
			if !found {
				return &sdk.Error{Code: sdk.ErrorNotFound, Message: "ACL entry was not found"}
			}
			return aclETagConflict(identity.ETag, current.ETag)
		}
		return nil
	}
	return invalid(errors.New("unsupported ACL operation"))
}

func (t *Transport) listACL(ctx context.Context, reportID string, output any) error {
	if _, err := t.getReportValue(ctx, reportID); err != nil {
		return err
	}
	rows, err := t.DB.QueryContext(ctx, `SELECT report_id,subject_type,subject_id,can_view,can_run,can_edit,can_publish,can_use_dql,etag FROM report_acl WHERE report_id=? ORDER BY subject_type,subject_id`, reportID)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	result := struct {
		Items []*sdk.ReportACL `json:"items"`
	}{}
	for rows.Next() {
		item := &sdk.ReportACL{}
		if err = rows.Scan(&item.ReportID, &item.SubjectType, &item.SubjectID, &item.CanView, &item.CanRun, &item.CanEdit, &item.CanPublish, &item.CanUseDQL, &item.ETag); err != nil {
			return internal(err)
		}
		result.Items = append(result.Items, item)
	}
	if err = rows.Err(); err != nil {
		return internal(err)
	}
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
		result, err := t.DB.ExecContext(ctx, `UPDATE report_acl SET can_view=?,can_run=?,can_edit=?,can_publish=?,can_use_dql=?,etag=etag+1 WHERE report_id=? AND subject_type=? AND subject_id=? AND etag=?`, value.CanView, value.CanRun, value.CanEdit, value.CanPublish, value.CanUseDQL, value.ReportID, value.SubjectType, value.SubjectID, value.ETag)
		if err != nil {
			return internal(err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			latest, latestFound, lookupErr := t.aclValue(ctx, value.ReportID, value.SubjectType, value.SubjectID)
			if lookupErr != nil {
				return internal(lookupErr)
			}
			if !latestFound {
				return &sdk.Error{Code: sdk.ErrorNotFound, Message: "ACL entry was not found"}
			}
			return aclETagConflict(value.ETag, latest.ETag)
		}
		value.ETag++
		return assign(output, &value)
	}
	_, err = t.DB.ExecContext(ctx, `INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run,can_edit,can_publish,can_use_dql,etag) VALUES (?,?,?,?,?,?,?,?,1)`, value.ReportID, value.SubjectType, value.SubjectID, value.CanView, value.CanRun, value.CanEdit, value.CanPublish, value.CanUseDQL)
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
	value := &sdk.ReportACL{}
	err := t.DB.QueryRowContext(ctx, `SELECT report_id,subject_type,subject_id,can_view,can_run,can_edit,can_publish,can_use_dql,etag FROM report_acl WHERE report_id=? AND subject_type=? AND subject_id=?`, reportID, subjectType, subjectID).
		Scan(&value.ReportID, &value.SubjectType, &value.SubjectID, &value.CanView, &value.CanRun, &value.CanEdit, &value.CanPublish, &value.CanUseDQL, &value.ETag)
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
