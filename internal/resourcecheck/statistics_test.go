package resourcecheck

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/studio/settings"
	api "google.golang.org/api/bigquery/v2"
	"testing"
)

func TestBudgetRejectsUnknownAndExceededEstimates(t *testing.T) {
	bytes, tables := int64(100), 1
	limits := &settings.ResourceLimits{MaxBytesProcessed: &bytes, MaxReferencedTables: &tables}
	job := &api.Job{Statistics: &api.JobStatistics{Query: &api.JobStatistics2{TotalBytesProcessed: 90, TotalBytesProcessedAccuracy: "PRECISE", ReferencedTables: []*api.TableReference{{ProjectId: "p", DatasetId: "d", TableId: "t"}}}}}
	require.NoError(t, CheckStatistics(job, limits))
	job.Statistics.Query.TotalBytesProcessed = 101
	require.ErrorContains(t, CheckStatistics(job, limits), "bytes processed exceed")
	job.Statistics.Query.TotalBytesProcessed = 90
	job.Statistics.Query.TotalBytesProcessedAccuracy = "LOWER_BOUND"
	require.ErrorContains(t, CheckStatistics(job, limits), "safe upper estimate")
	job.Statistics.Query.TotalBytesProcessedAccuracy = "PRECISE"
	job.Statistics.Query.ReferencedTables = append(job.Statistics.Query.ReferencedTables, &api.TableReference{ProjectId: "p", DatasetId: "d", TableId: "other"})
	require.ErrorContains(t, CheckStatistics(job, limits), "referenced tables exceed")
	require.Error(t, CheckStatistics(&api.Job{}, limits))
}
