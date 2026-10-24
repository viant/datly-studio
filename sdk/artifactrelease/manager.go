// Package artifactrelease defines the host-facing named release manager seam.
// Each implementation derives authority from the verified request context;
// management DTOs never carry trusted tenant, account or workspace identity.
package artifactrelease

import (
	"context"

	contract "github.com/viant/forge/backend/artifactrelease"
)

type Manager interface {
	List(context.Context, contract.ListInput) (contract.ListOutput, error)
	Candidates(context.Context, contract.CandidatesInput) (contract.CandidatesOutput, error)
	Publish(context.Context, contract.PublishInput) (contract.Result, error)
	Promote(context.Context, contract.PromoteInput) (contract.Result, error)
	History(context.Context, contract.HistoryInput) (contract.HistoryOutput, error)
}
