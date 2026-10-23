package writer

import (
	"context"
	"testing"

	"github.com/viant/datly-studio/internal/resourceartifact"
	xhandler "github.com/viant/xdatly/handler"
)

func TestPublicFileWriterRejectsCompiledNamespaceBeforeAuthorization(t *testing.T) {
	resource := resourceartifact.Namespace
	report := "report"
	value := &ReportResourceFile{ReportId: &report, Namespace: &resource}
	if err := (&FileRules{}).Validate(context.Background(), value, xhandler.LifecycleContext[ReportResourceFile, xhandler.NoParent, Output]{}); err == nil {
		t.Fatal("public writer accepted internal compiled namespace")
	}
}
