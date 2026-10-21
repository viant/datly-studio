package audit

import (
	"context"
	"fmt"
	"github.com/viant/datly-studio/internal/auditstore"
	"strings"
	"time"
)

const ExportMaxBytes = 8 << 20
const ExportMaxRows = 10000

type Export struct {
	Filename  string `json:"filename"`
	MediaType string `json:"mediaType"`
	Archive   []byte `json:"archive"`
	Count     int    `json:"count"`
	Truncated bool   `json:"truncated"`
	NextID    int64  `json:"nextId,omitempty"`
}

// ExportCSV pages through the authorized predicate reader and delegates CSV
// serialization to Datly's native output plan. Only event metadata is exported.
func (s *Service) ExportCSV(ctx context.Context, filter Filter, maxRows int) (*Export, error) {
	if maxRows <= 0 || maxRows > ExportMaxRows {
		maxRows = ExportMaxRows
	}
	filter.Limit = 100
	result := &Export{Filename: "audit-" + time.Now().UTC().Format("20060102-150405") + ".csv", MediaType: "text/csv; charset=utf-8"}
	rows := make([]auditstore.CSVEvent, 0, 100)
	estimatedBytes := 1024 // room for the fixed header
	finish := func() (*Export, error) {
		data, err := auditstore.EncodeCSV(ctx, rows)
		if err != nil {
			return nil, err
		}
		if len(data) > ExportMaxBytes {
			return nil, fmt.Errorf("audit CSV exceeds export size limit")
		}
		result.Archive = data
		return result, nil
	}
	var lastID int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := s.List(ctx, filter)
		if err != nil {
			return nil, err
		}
		for index, event := range page.Items {
			row := auditstore.CSVEvent{ID: event.ID, NamespaceID: csvCell(event.NamespaceID), ComponentID: csvCell(event.ComponentID), ToolName: csvCell(event.ToolName), StartedAt: event.StartedAt.UTC().Format(time.RFC3339Nano), DurationMS: event.DurationMS, Status: csvCell(event.Status), Subject: csvCell(event.Subject), ClientIP: csvCell(event.ClientIP), ResponseTruncated: event.ResponseTruncated}
			// Bound memory before native encoding, allowing every string byte to be escaped.
			upperBound := 256
			for _, value := range []string{row.NamespaceID, row.ComponentID, row.ToolName, row.StartedAt, row.Status, row.Subject, row.ClientIP} {
				upperBound += 2 * len(value)
			}
			if estimatedBytes+upperBound > ExportMaxBytes {
				if result.Count == 0 {
					return nil, fmt.Errorf("audit event exceeds CSV export size limit")
				}
				result.Truncated = true
				result.NextID = lastID
				return finish()
			}
			estimatedBytes += upperBound
			rows = append(rows, row)
			result.Count++
			lastID = event.ID
			if result.Count == maxRows {
				result.Truncated = index+1 < len(page.Items) || page.NextID > 0
				if result.Truncated {
					result.NextID = lastID
				}
				return finish()
			}
		}
		if page.NextID == 0 {
			return finish()
		}
		filter.BeforeID = page.NextID
	}
}

// Keep untrusted metadata literal when a spreadsheet opens the exported file.
func csvCell(value string) string {
	value = strings.ToValidUTF8(value, "�")
	trimmed := strings.TrimSpace(value)
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) || strings.HasPrefix(value, "\t") || strings.HasPrefix(value, "\r") {
		return "'" + value
	}
	return value
}
