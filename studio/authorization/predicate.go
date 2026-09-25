// Package authorization contains Studio's application-owned Datly predicate
// handlers. They consume only an explicitly bound, verified JWT input.
package authorization

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly/connector"
	xpredicate "github.com/viant/xdatly/predicate"
	xresponse "github.com/viant/xdatly/response"
)

// LinkedTypes supplies explicit Datly v1 type authority for the verified JWT
// value and every predicate handler imported by Studio DQL.
type LinkedTypes struct {
	Claims                     jwt.Claims
	ConnectorRead              ConnectorRead
	ConnectorEdit              ConnectorEdit
	AuthorizationPredicateRead AuthorizationPredicateRead
	AuthorizationPredicateEdit AuthorizationPredicateEdit
	ReportRead                 ReportRead
	ReportEdit                 ReportEdit
	ReportPublish              ReportPublish
	ReportVersionRead          ReportVersionRead
	ReportVersionEdit          ReportVersionEdit
	ReportViewRead             ReportViewRead
	ReportParameterRead        ReportParameterRead
	ReportParameterEdit        ReportParameterEdit
	ReportCubeRead             ReportCubeRead
	ReportCubeEdit             ReportCubeEdit
	ReportMCPRead              ReportMCPRead
	ReportMCPEdit              ReportMCPEdit
	ReportResourceFileRead     ReportResourceFileRead
	ReportResourceFileEdit     ReportResourceFileEdit
	ReportResourceFolderRead   ReportResourceFolderRead
	ReportResourceFolderEdit   ReportResourceFolderEdit
	ReportSkillRead            ReportSkillRead
	ReportSkillEdit            ReportSkillEdit
	PublicationRead            PublicationRead
	PublicationEdit            PublicationEdit
	PublicationEventRead       PublicationEventRead
	ACLRead                    ACLRead
	ACLEdit                    ACLEdit
	RuntimeRead                RuntimeRead
	RuntimeEdit                RuntimeEdit
	WarmupRead                 WarmupRead
	SessionRead                SessionRead
	SessionRevoke              SessionRevoke
}

// StudioAuthorizationDatlyType keeps the authorization contract linked for
// package scanning without central registration or init side effects.
func StudioAuthorizationDatlyType() reflect.Type { return reflect.TypeOf((*LinkedTypes)(nil)).Elem() }

var StudioAuthorizationDatlyLinkedType = StudioAuthorizationDatlyType()

// DatlyPredicateHandlerTypes is a type-link anchor, not a registry. Datly
// discovers the handlers by scanning this package; retaining their runtime
// type descriptors lets that scan resolve DQL names in a linked executable.
var DatlyPredicateHandlerTypes = []reflect.Type{
	reflect.TypeFor[ConnectorRead](), reflect.TypeFor[ConnectorEdit](), reflect.TypeFor[ReportRead](), reflect.TypeFor[ReportEdit](), reflect.TypeFor[ReportPublish](),
	reflect.TypeFor[AuthorizationPredicateRead](), reflect.TypeFor[AuthorizationPredicateEdit](),
	reflect.TypeFor[ReportVersionRead](), reflect.TypeFor[ReportVersionEdit](), reflect.TypeFor[ReportViewRead](), reflect.TypeFor[ReportParameterRead](), reflect.TypeFor[ReportParameterEdit](),
	reflect.TypeFor[ReportCubeRead](), reflect.TypeFor[ReportCubeEdit](), reflect.TypeFor[ReportMCPRead](), reflect.TypeFor[ReportMCPEdit](),
	reflect.TypeFor[ReportResourceFileRead](), reflect.TypeFor[ReportResourceFileEdit](), reflect.TypeFor[ReportResourceFolderRead](), reflect.TypeFor[ReportResourceFolderEdit](),
	reflect.TypeFor[ReportSkillRead](), reflect.TypeFor[ReportSkillEdit](), reflect.TypeFor[PublicationRead](), reflect.TypeFor[PublicationEdit](), reflect.TypeFor[PublicationEventRead](),
	reflect.TypeFor[ACLRead](), reflect.TypeFor[ACLEdit](), reflect.TypeFor[RuntimeRead](), reflect.TypeFor[RuntimeEdit](), reflect.TypeFor[WarmupRead](),
	reflect.TypeFor[SessionRead](), reflect.TypeFor[SessionRevoke](),
}

const (
	permissionView    = "can_view"
	permissionEdit    = "can_edit"
	permissionPublish = "can_publish"
)

// InputBinding is embedded structurally in every handler so Datly binds the
// component's declared input. It intentionally does not read request context.
type InputBinding struct {
	Input any `bind:"kind=input,required"`
}

type ConnectorRead struct{ InputBinding }
type ConnectorEdit struct{ InputBinding }
type AuthorizationPredicateRead struct{ InputBinding }
type AuthorizationPredicateEdit struct{ InputBinding }
type ReportRead struct{ InputBinding }
type ReportEdit struct{ InputBinding }
type ReportPublish struct{ InputBinding }
type ReportVersionRead struct{ InputBinding }
type ReportVersionEdit struct{ InputBinding }
type ReportViewRead struct{ InputBinding }
type ReportParameterRead struct{ InputBinding }
type ReportParameterEdit struct{ InputBinding }
type ReportCubeRead struct{ InputBinding }
type ReportCubeEdit struct{ InputBinding }
type ReportMCPRead struct{ InputBinding }
type ReportMCPEdit struct{ InputBinding }
type ReportResourceFileRead struct{ InputBinding }
type ReportResourceFileEdit struct{ InputBinding }
type ReportResourceFolderRead struct{ InputBinding }
type ReportResourceFolderEdit struct{ InputBinding }
type ReportSkillRead struct{ InputBinding }
type ReportSkillEdit struct{ InputBinding }
type PublicationRead struct{ InputBinding }
type PublicationEdit struct{ InputBinding }
type PublicationEventRead struct{ InputBinding }
type ACLRead struct{ InputBinding }
type ACLEdit struct{ InputBinding }
type RuntimeRead struct{ InputBinding }
type RuntimeEdit struct{ InputBinding }
type WarmupRead struct{ InputBinding }
type SessionRead struct{ InputBinding }
type SessionRevoke struct{ InputBinding }

func (p *ConnectorRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return connectorCriteria(ctx, p.Input, "connectors.name", "connectors.owner_id", permissionView)
}
func (p *ConnectorEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	principal, err := subject(ctx, p.Input)
	if err != nil {
		return nil, err
	}
	if err := requireOwnedRows(p.Input, principal, "Connectors"); err != nil {
		return nil, err
	}
	return connectorCriteriaForSubject(principal, "connector.name", "connector.owner_id", permissionEdit), nil
}
func (p *AuthorizationPredicateRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return globalCriteria(ctx, p.Input, permissionPublish)
}
func (p *AuthorizationPredicateEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return globalCriteria(ctx, p.Input, permissionPublish)
}
func (p *ReportRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "reports.id", permissionView)
}
func (p *ReportEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	principal, err := subject(ctx, p.Input)
	if err != nil {
		return nil, err
	}
	if err := requireOwnedRows(p.Input, principal, "Reports"); err != nil {
		return nil, err
	}
	return reportCriteriaForSubject(principal, "report.id", permissionEdit), nil
}
func (p *ReportPublish) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "report.id", permissionPublish)
}
func (p *ReportVersionRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "v.report_id", permissionView)
}
func (p *ReportVersionEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "report_version.report_id", permissionEdit)
}
func (p *ReportViewRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "v.report_id", permissionView)
}
func (p *ReportParameterRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "p.report_id", permissionView)
}
func (p *ReportParameterEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "parameter.report_id", permissionEdit)
}
func (p *ReportCubeRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "c.report_id", permissionView)
}
func (p *ReportCubeEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "cube_config.report_id", permissionEdit)
}
func (p *ReportMCPRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "e.report_id", permissionView)
}
func (p *ReportMCPEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "exposure.report_id", permissionEdit)
}
func (p *ReportResourceFileRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "f.report_id", permissionView)
}
func (p *ReportResourceFileEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "resource_file.report_id", permissionEdit)
}
func (p *ReportResourceFolderRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "f.report_id", permissionView)
}
func (p *ReportResourceFolderEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "resource_folder.report_id", permissionEdit)
}
func (p *ReportSkillRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "s.report_id", permissionView)
}
func (p *ReportSkillEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "skill_root.report_id", permissionEdit)
}
func (p *PublicationRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "p.report_id", permissionView)
}
func (p *PublicationEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "publication.report_id", permissionPublish)
}

// PublicationEventRead scopes the immutable lifecycle audit to the report
// owner's tenant. Delegated publisher grants never make an owner's event
// history visible across owner boundaries.
func (p *PublicationEventRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	subject, err := subject(ctx, p.Input)
	if err != nil {
		return nil, err
	}
	return &xpredicate.Criteria{Expression: "event.owner_id = ?", Placeholders: []any{subject}}, nil
}
func (p *ACLRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "a.report_id", permissionEdit)
}
func (p *ACLEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "report_acl.report_id", permissionPublish)
}
func (p *RuntimeRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return globalCriteria(ctx, p.Input, permissionPublish)
}
func (p *RuntimeEdit) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return globalCriteria(ctx, p.Input, permissionPublish)
}
func (p *WarmupRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	return reportCriteria(ctx, p.Input, "w.report_id", permissionPublish)
}
func (p *SessionRead) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	principal, err := subject(ctx, p.Input)
	if err != nil {
		return nil, err
	}
	return &xpredicate.Criteria{Expression: `s.subject_id = ?`, Placeholders: []any{principal}}, nil
}
func (p *SessionRevoke) Compute(ctx context.Context, _ any) (*xpredicate.Criteria, error) {
	principal, err := subject(ctx, p.Input)
	if err != nil {
		return nil, err
	}
	return &xpredicate.Criteria{Expression: `session.subject_id = ?`, Placeholders: []any{principal}}, nil
}

func connectorCriteria(ctx context.Context, input any, nameColumn, ownerColumn, permission string) (*xpredicate.Criteria, error) {
	subject, err := subject(ctx, input)
	if err != nil {
		return nil, err
	}
	return connectorCriteriaForSubject(subject, nameColumn, ownerColumn, permission), nil
}

func connectorCriteriaForSubject(subject, nameColumn, ownerColumn, permission string) *xpredicate.Criteria {
	return &xpredicate.Criteria{Expression: `(` + ownerColumn + ` = ? OR EXISTS (
SELECT 1 FROM reports studio_auth_report
JOIN report_acl studio_auth_acl ON studio_auth_acl.report_id = studio_auth_report.id
WHERE studio_auth_report.default_connector_name = ` + nameColumn + `
  AND studio_auth_acl.subject_type = 'user'
  AND studio_auth_acl.subject_id = ?
	  AND studio_auth_acl.` + permission + ` = TRUE))`, Placeholders: []any{subject, subject}}
}

func reportCriteria(ctx context.Context, input any, reportIDColumn, permission string) (*xpredicate.Criteria, error) {
	subject, err := subject(ctx, input)
	if err != nil {
		return nil, err
	}
	return reportCriteriaForSubject(subject, reportIDColumn, permission), nil
}

func reportCriteriaForSubject(subject, reportIDColumn, permission string) *xpredicate.Criteria {
	return &xpredicate.Criteria{Expression: `(EXISTS (
SELECT 1 FROM reports studio_auth_owner
WHERE studio_auth_owner.id = ` + reportIDColumn + ` AND studio_auth_owner.owner_id = ?
) OR EXISTS (
SELECT 1 FROM report_acl studio_auth_acl
WHERE studio_auth_acl.report_id = ` + reportIDColumn + `
  AND studio_auth_acl.subject_type = 'user'
  AND studio_auth_acl.subject_id = ?
	  AND studio_auth_acl.` + permission + ` = TRUE))`, Placeholders: []any{subject, subject}}
}

func globalCriteria(ctx context.Context, input any, permission string) (*xpredicate.Criteria, error) {
	subject, err := subject(ctx, input)
	if err != nil {
		return nil, err
	}
	return &xpredicate.Criteria{Expression: `EXISTS (
SELECT 1 FROM report_acl studio_auth_acl
WHERE studio_auth_acl.subject_type = 'user'
  AND studio_auth_acl.subject_id = ?
  AND studio_auth_acl.` + permission + ` = TRUE)`, Placeholders: []any{subject}}, nil
}

func requireOwnedRows(input any, principal, collection string) error {
	value := indirect(reflect.ValueOf(input))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return forbidden("authorization input is unavailable")
	}
	rows := indirect(value.FieldByName(collection))
	if !rows.IsValid() {
		return nil
	}
	if rows.Kind() != reflect.Slice && rows.Kind() != reflect.Array {
		return forbidden("authorization rows are invalid")
	}
	for index := 0; index < rows.Len(); index++ {
		row := indirect(rows.Index(index))
		if !row.IsValid() || row.Kind() != reflect.Struct {
			continue
		}
		owner := stringField(row, "OwnerId", "OwnerID")
		if owner != "" && owner != principal {
			return forbidden("cannot create or transfer another principal's resource")
		}
	}
	return nil
}

func indirect(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func stringField(value reflect.Value, names ...string) string {
	for _, name := range names {
		field := indirect(value.FieldByName(name))
		if field.IsValid() && field.Kind() == reflect.String {
			return strings.TrimSpace(field.String())
		}
	}
	return ""
}

func subject(ctx context.Context, input any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	claims := claimsFromInput(input)
	if claims == nil {
		return "", forbidden("verified JWT claims are required")
	}
	return principalFromClaims(claims)
}

// AuthorizeReport verifies a report-scoped write through the same owner/ACL
// policy used by reader predicates. Lifecycle hooks call it for new child rows,
// where a SQL predicate has no previous row to constrain.
func AuthorizeReport(ctx context.Context, provider connector.Provider, claims *jwt.Claims, reportID, permission string) error {
	if provider == nil {
		return forbidden("authorization connector is unavailable")
	}
	principal, err := principalFromClaims(claims)
	if err != nil {
		return err
	}
	if strings.TrimSpace(reportID) == "" {
		return forbidden("report identity is required")
	}
	if permission != permissionView && permission != permissionEdit && permission != permissionPublish {
		return forbidden("authorization permission is invalid")
	}
	grants, err := readGrants(ctx, provider, principal, reportID, "", false)
	if err != nil {
		return err
	}
	for _, grant := range grants {
		switch permission {
		case permissionView:
			if grant.CanView {
				return nil
			}
		case permissionEdit:
			if grant.CanEdit {
				return nil
			}
		case permissionPublish:
			if grant.CanPublish {
				return nil
			}
		}
	}
	return forbidden("report authorization is required")
}

// AuthorizeGlobal requires a publisher permission for lifecycle operations that
// do not belong to one specific report, such as runtime generation changes.
func AuthorizeGlobal(ctx context.Context, provider connector.Provider, claims *jwt.Claims, permission string) error {
	if provider == nil {
		return forbidden("authorization connector is unavailable")
	}
	principal, err := principalFromClaims(claims)
	if err != nil {
		return err
	}
	if permission != permissionPublish {
		return forbidden("authorization permission is invalid")
	}
	grants, err := readGrants(ctx, provider, principal, "", "acl", true)
	if err != nil {
		return err
	}
	if len(grants) == 0 {
		return forbidden("publisher authorization is required")
	}
	return nil
}

// AuthorizeOwner verifies a simple owner/subject invariant after JWT decoding.
// It is intended for writer lifecycle validation, not SQL predicate expansion.
func AuthorizeOwner(claims *jwt.Claims, ownerID string) error {
	principal, err := principalFromClaims(claims)
	if err != nil {
		return err
	}
	if strings.TrimSpace(ownerID) == "" || strings.TrimSpace(ownerID) != principal {
		return forbidden("resource owner authorization is required")
	}
	return nil
}

func principalFromClaims(claims *jwt.Claims) (string, error) {
	if claims == nil {
		return "", forbidden("verified JWT claims are required")
	}
	if subject := strings.TrimSpace(claims.Subject); subject != "" {
		return subject, nil
	}
	return "", forbidden("JWT subject is required")
}

func claimsFromInput(input any) *jwt.Claims {
	value := reflect.ValueOf(input)
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return nil
	}
	for _, name := range []string{"Jwt", "JWT"} {
		field := value.FieldByName(name)
		if !field.IsValid() || !field.CanInterface() {
			continue
		}
		if claims, ok := field.Interface().(*jwt.Claims); ok {
			return claims
		}
	}
	return nil
}

func forbidden(message string) error { return &xresponse.Error{Code: 403, Cause: errors.New(message)} }
