package preview

import (
	"github.com/viant/datly-studio/sdk"
	dexec "github.com/viant/datly/exec"
)

// warmupTargetEvidence projects initialized native facts, never authored storage
// settings.
// In particular, Work entries retain their containing scheduled policy and are
// never promoted to scheduled targets or used to change execution counts.
func warmupTargetEvidence(rootView string, targets []dexec.ReaderWarmupTarget) sdk.WarmupTarget {
	result := sdk.WarmupTarget{View: rootView}
	for _, target := range targets {
		scheduled := sdk.WarmupScheduledTarget{View: target.View, CacheName: target.CacheName, CacheProvider: target.Backend}
		if settings := target.Settings; settings != nil {
			scheduled.PolicyName = settings.Name
			scheduled.ConnectorName = settings.Connector
			scheduled.IndexColumn = settings.IndexColumn
			scheduled.IndexParameter = settings.IndexParameter
			scheduled.IndexMeta = settings.IndexMeta
		}
		for _, work := range target.Work {
			item := sdk.WarmupWork{View: work.View, CacheName: work.CacheName, CacheProvider: work.Backend}
			if settings := work.Settings; settings != nil {
				item.PolicyName = settings.Name
				item.ConnectorName = settings.Connector
				item.IndexColumn = settings.IndexColumn
				item.IndexParameter = settings.IndexParameter
			}
			scheduled.Work = append(scheduled.Work, item)
		}
		result.ScheduledTargets = append(result.ScheduledTargets, scheduled)
	}
	// Existing consumers have only one cache slot. Populate it only when it
	// unambiguously describes all population; mixed or expanded work stays in
	// the hierarchy. A known logical name is independent of backend knowledge.
	if len(result.ScheduledTargets) == 1 && len(result.ScheduledTargets[0].Work) == 1 {
		target := result.ScheduledTargets[0]
		work := target.Work[0]
		result.CacheName, result.CacheProvider = work.CacheName, work.CacheProvider
		result.ConnectorName = target.ConnectorName
		result.IndexColumn, result.IndexParameter = target.IndexColumn, target.IndexParameter
	}
	return result
}
