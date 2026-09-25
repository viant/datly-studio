package sqltransport

import (
	"context"

	"github.com/viant/datly-studio/internal/reportcapability"
	stored "github.com/viant/datly-studio/studio/reports/store_capabilities"
)

func (t *Transport) readReportCapability(ctx context.Context, reportID, subject string) (*stored.ReportCapability, error) {
	reader, err := t.reportCapabilityReader()
	if err != nil {
		return nil, err
	}
	return reader.Read(ctx, reportID, subject)
}

func (t *Transport) reportCapabilityReader() (*reportcapability.Reader, error) {
	t.capabilityReaderMu.Lock()
	defer t.capabilityReaderMu.Unlock()
	if t.capabilityReader != nil {
		return t.capabilityReader, nil
	}
	reader, err := reportcapability.New(t.DB)
	if err != nil {
		return nil, err
	}
	t.capabilityReader = reader
	return t.capabilityReader, nil
}
