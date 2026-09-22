package authorization

import (
	"testing"

	"github.com/viant/datly/bootstrap"
)

func TestPredicateHandlersAreLinkedForDatlyPackageScanning(t *testing.T) {
	for _, name := range []string{
		"ConnectorRead", "ConnectorEdit", "ReportRead", "ReportEdit", "ReportPublish",
		"ReportVersionRead", "ReportVersionEdit", "ReportViewRead", "ReportParameterRead",
		"ReportParameterEdit", "ReportCubeRead", "ReportCubeEdit", "ReportMCPRead",
		"ReportMCPEdit", "ReportResourceFileRead", "ReportResourceFileEdit", "ReportResourceFolderRead",
		"ReportResourceFolderEdit", "ReportSkillRead", "ReportSkillEdit", "PublicationRead",
		"PublicationEdit", "ACLRead", "ACLEdit", "RuntimeRead", "RuntimeEdit", "WarmupRead", "SessionRead", "SessionRevoke",
	} {
		if bootstrap.LinkedHolder(nil, "github.com/viant/datly-studio/studio/authorization", name) == nil {
			t.Fatalf("authorization handler %s is not linked", name)
		}
	}
}
