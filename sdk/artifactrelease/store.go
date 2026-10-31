package artifactrelease

import (
	"context"
	"github.com/viant/authz"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	contract "github.com/viant/forge/backend/artifactrelease"
)

// ReleaseStore is the persistence contract required by the release controller.
// Hosts inject its native component implementation with verified request scope.
type ReleaseStore interface {
	contract.ChannelStore
	CreateChannel(context.Context, contract.Channel) error
	ListChannels(context.Context) ([]contract.Channel, error)
	History(context.Context, contract.Key) ([]releasevalue.HistoryEntry, error)
	BundleBindings(context.Context, contract.Bundle) ([]releasevalue.ComponentBinding, error)
	ActiveDependencies(context.Context, contract.Key) ([]authz.NamedDependency, error)
	PendingDependencies(context.Context, string) ([]authz.NamedDependency, error)
	StageRevisionWithBindingsAndDependencies(context.Context, contract.Channel, contract.Channel, contract.Bundle, []releasevalue.ComponentBinding, []authz.NamedDependency) (string, error)
	FailRevision(context.Context, contract.Key, uint64, string, string) error
}
