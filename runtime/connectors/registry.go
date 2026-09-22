package connectors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/viant/datly-studio/sdk"
)

type RegistrySnapshot struct {
	Connectors map[string]*sdk.Connector
	Ordered    []*sdk.Connector
	Revision   string
	CreatedAt  time.Time
}

type RegistryBuilder struct {
	now func() time.Time
}

type RegistryOption func(*RegistryBuilder)

func WithNow(fn func() time.Time) RegistryOption {
	return func(builder *RegistryBuilder) {
		if fn != nil {
			builder.now = fn
		}
	}
}

func NewRegistryBuilder(options ...RegistryOption) *RegistryBuilder {
	result := &RegistryBuilder{
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, option := range options {
		option(result)
	}
	return result
}

func (b *RegistryBuilder) Build(_ context.Context, connectors []*sdk.Connector) (*RegistrySnapshot, error) {
	index := make(map[string]*sdk.Connector, len(connectors))
	ordered := make([]*sdk.Connector, 0, len(connectors))
	for _, connector := range connectors {
		if connector == nil {
			return nil, fmt.Errorf("connector snapshot: nil connector")
		}
		if _, exists := index[connector.Name]; exists {
			return nil, fmt.Errorf("connector snapshot: duplicate connector %q", connector.Name)
		}
		cloned := cloneConnector(connector)
		index[cloned.Name] = cloned
		ordered = append(ordered, cloned)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].UpdatedAt.Equal(ordered[j].UpdatedAt) {
			return ordered[i].Name < ordered[j].Name
		}
		return ordered[i].UpdatedAt.After(ordered[j].UpdatedAt)
	})

	revision, err := computeRevision(ordered)
	if err != nil {
		return nil, err
	}
	return &RegistrySnapshot{
		Connectors: index,
		Ordered:    ordered,
		Revision:   revision,
		CreatedAt:  b.now(),
	}, nil
}

func (s *RegistrySnapshot) Get(name string) (*sdk.Connector, bool) {
	if s == nil {
		return nil, false
	}
	connector, ok := s.Connectors[name]
	return connector, ok
}

func (s *RegistrySnapshot) List() []*sdk.Connector {
	if s == nil {
		return nil
	}
	result := make([]*sdk.Connector, len(s.Ordered))
	copy(result, s.Ordered)
	return result
}

func cloneConnector(connector *sdk.Connector) *sdk.Connector {
	cloned := *connector
	cloned.Options = append([]byte(nil), connector.Options...)
	if connector.LastTestedAt != nil {
		value := *connector.LastTestedAt
		cloned.LastTestedAt = &value
	}
	return &cloned
}

func computeRevision(connectors []*sdk.Connector) (string, error) {
	payload := make([]map[string]any, 0, len(connectors))
	for _, connector := range connectors {
		payload = append(payload, map[string]any{
			"name":        connector.Name,
			"driver":      connector.Driver,
			"dsnTemplate": connector.DSNTemplate,
			"ownerId":     connector.OwnerID,
			"status":      connector.Status,
			"secretRef":   connector.SecretRef,
			"etag":        connector.ETag,
			"updatedAt":   connector.UpdatedAt.UTC().Format(time.RFC3339Nano),
			"options":     connector.Options,
		})
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("connector snapshot revision: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
