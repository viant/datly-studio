package writer

import (
	"context"
	"strings"
	"testing"

	xhandler "github.com/viant/xdatly/handler"
)

func TestACLRulesRequireUserJWTSubject(t *testing.T) {
	reportID, role, subject := "report-1", "role", "analyst"
	rules := &ACLRules{}
	err := rules.Validate(context.Background(), &ReportACL{ReportId: &reportID, SubjectType: &role, SubjectId: &subject}, xhandler.LifecycleContext[ReportACL, xhandler.NoParent, Output]{})
	if err == nil || !strings.Contains(err.Error(), "subject type must be user") {
		t.Fatalf("role ACL validation error = %v", err)
	}
}

func TestACLRulesRequireSubjectBeforeAuthorization(t *testing.T) {
	reportID, kind := "report-1", "user"
	rules := &ACLRules{}
	err := rules.Validate(context.Background(), &ReportACL{ReportId: &reportID, SubjectType: &kind}, xhandler.LifecycleContext[ReportACL, xhandler.NoParent, Output]{})
	if err == nil || !strings.Contains(err.Error(), "verified JWT sub") {
		t.Fatalf("empty subject validation error = %v", err)
	}
}
