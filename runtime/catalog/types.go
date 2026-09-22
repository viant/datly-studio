package catalog

import "github.com/viant/datly-studio/sdk"

// PublishedReport is the canonical runtime catalog record. It deliberately
// uses the SDK DTOs rather than the retired control/component model.
type PublishedReport struct {
	Report      *sdk.Report
	Version     *sdk.ReportVersion
	Publication *sdk.Publication
}
