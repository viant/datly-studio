package main

import (
	"testing"

	"github.com/viant/datly/bootstrap"
)

func TestExecutableLinksAuthorizationHandlers(t *testing.T) {
	for _, name := range []string{"ConnectorRead", "ReportRead", "RuntimeEdit"} {
		if bootstrap.LinkedHolder(nil, "github.com/viant/datly-studio/studio/authorization", name) == nil {
			t.Fatalf("Datly executable does not link %s", name)
		}
	}
}
