package writer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

// PolicyRules enforces the resource policy activation contract on the head
// row. The supplied head revision is the caller's expected current revision
// (0 for bootstrap provisioning); the committed head advances by exactly one
// and exactly one new history row records that revision. Datly's universal
// writer applies the head and its history child in one managed transaction:
// the head's concurrency token rejects a stale expected revision, the history
// primary key rejects a concurrent activation of the same revision, and a
// failed history insert rolls the head update back. Every rule below runs
// before any row is written.
type PolicyRules struct {
	Input *Input `bind:"kind=input,required"`
}

// PolicyRulesHooks keeps the lifecycle type reachable for Datly package
// scanning without registration or init side effects.
var PolicyRulesHooks = new(PolicyRules)

func PolicyRulesDatlyType() reflect.Type { return reflect.TypeOf((*PolicyRules)(nil)).Elem() }

var PolicyRulesDatlyLinkedType = PolicyRulesDatlyType()

const policyEntity = "policy"

func conflict(reason string) error {
	return &xhandler.Conflict{Entity: policyEntity, Field: "revision", Reason: reason}
}

func invalid(reason string) error {
	return &xresponse.Error{Code: 422, Cause: errors.New(reason)}
}

// Init turns the caller's expected revision into the next committed revision.
// The writer captured the supplied value as the concurrency token before Init
// ran, so a stale expectation still fails against the persisted head.
func (hooks *PolicyRules) Init(_ context.Context, head *ResourcePolicyHead, _ xhandler.LifecycleContext[ResourcePolicyHead, xhandler.NoParent, Output]) error {
	if head == nil {
		return invalid("resource policy head is required")
	}
	if head.Revision == nil || *head.Revision < 0 {
		return conflict("expected revision is required")
	}
	next := *head.Revision + 1
	head.Revision = &next
	if head.Has == nil {
		head.Has = &ResourcePolicyHeadHas{}
	}
	head.Has.Revision = true
	for _, row := range head.History {
		if row == nil {
			continue
		}
		if row.Has == nil {
			row.Has = &ResourcePolicyRevisionHas{}
		}
		if row.TenantId == "" {
			row.TenantId, row.Has.TenantId = head.TenantId, true
		}
		if row.ResourceKind == "" {
			row.ResourceKind, row.Has.ResourceKind = head.ResourceKind, true
		}
		if row.ResourceId == "" {
			row.ResourceId, row.Has.ResourceId = head.ResourceId, true
		}
		if row.ResourceVersion == "" {
			row.ResourceVersion, row.Has.ResourceVersion = head.ResourceVersion, true
		}
		if row.Revision == nil {
			revision := next
			row.Revision, row.Has.Revision = &revision, true
		}
		if row.OccurredAt != nil && !row.OccurredAt.IsZero() && strings.TrimSpace(row.ActorId) != "" {
			occurred, actor := *row.OccurredAt, row.ActorId
			row.SetCreatedAt(&occurred)
			row.SetCreatedBy(&actor)
			row.SetUpdatedAt(&occurred)
			row.SetUpdatedBy(&actor)
		}
	}
	if len(head.History) == 1 && head.History[0] != nil && head.History[0].OccurredAt != nil &&
		!head.History[0].OccurredAt.IsZero() && strings.TrimSpace(head.History[0].ActorId) != "" {
		row := head.History[0]
		occurred, actor := *row.OccurredAt, row.ActorId
		head.SetUpdatedAt(&occurred)
		head.SetUpdatedBy(&actor)
		if current := hooks.current(head); current != nil {
			if current.CreatedAt == nil || current.CreatedAt.IsZero() || current.CreatedBy == nil || strings.TrimSpace(*current.CreatedBy) == "" {
				return invalid("resource policy existing audit fields are incomplete")
			}
			created, creator := *current.CreatedAt, *current.CreatedBy
			head.SetCreatedAt(&created)
			head.SetCreatedBy(&creator)
		} else {
			head.SetCreatedAt(&occurred)
			head.SetCreatedBy(&actor)
		}
	}
	return nil
}

// Validate denies before mutation: bootstrap cannot overwrite an existing
// policy, a replacement needs an existing head at exactly the expected
// revision, and the activation must carry one complete audit record.
func (hooks *PolicyRules) Validate(_ context.Context, head *ResourcePolicyHead, _ xhandler.LifecycleContext[ResourcePolicyHead, xhandler.NoParent, Output]) error {
	if head == nil || head.Revision == nil {
		return invalid("resource policy head is required")
	}
	for name, value := range map[string]string{"tenant_id": head.TenantId, "resource_kind": head.ResourceKind, "resource_id": head.ResourceId, "resource_version": head.ResourceVersion} {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
			return invalid("resource policy " + name + " is required")
		}
	}
	next := *head.Revision
	expected := next - 1
	if expected < 0 {
		return conflict("expected revision is required")
	}
	current := hooks.current(head)
	switch {
	case current == nil && expected != 0:
		return conflict("resource policy does not exist at the expected revision")
	case current != nil && expected == 0:
		return conflict("resource policy is already provisioned")
	case current != nil && (current.Revision == nil || *current.Revision != expected):
		return conflict("resource policy revision is stale")
	}
	if len(head.History) != 1 || head.History[0] == nil {
		return invalid("resource policy activation requires exactly one history record")
	}
	row := head.History[0]
	if row.TenantId != head.TenantId || row.ResourceKind != head.ResourceKind || row.ResourceId != head.ResourceId || row.ResourceVersion != head.ResourceVersion {
		return invalid("resource policy history must belong to the head")
	}
	if row.Revision == nil || *row.Revision != next {
		return invalid("resource policy history must record the committed revision")
	}
	if strings.TrimSpace(row.ActorId) == "" {
		return invalid("resource policy actor is required")
	}
	if row.OccurredAt == nil || row.OccurredAt.IsZero() {
		return invalid("resource policy occurrence time is required")
	}
	if head.CreatedAt == nil || head.CreatedAt.IsZero() || head.CreatedBy == nil || strings.TrimSpace(*head.CreatedBy) == "" ||
		head.UpdatedAt == nil || head.UpdatedAt.IsZero() || head.UpdatedBy == nil || strings.TrimSpace(*head.UpdatedBy) == "" ||
		row.CreatedAt == nil || row.CreatedAt.IsZero() || row.CreatedBy == nil || strings.TrimSpace(*row.CreatedBy) == "" ||
		row.UpdatedAt == nil || row.UpdatedAt.IsZero() || row.UpdatedBy == nil || strings.TrimSpace(*row.UpdatedBy) == "" {
		return invalid("resource policy audit fields are required")
	}
	trimmed := bytes.TrimSpace(row.PoliciesJson)
	if len(trimmed) == 0 || trimmed[0] != '{' || !json.Valid(trimmed) {
		return invalid("resource policy document must be a JSON object")
	}
	for _, existing := range hooks.history(head) {
		if existing != nil && existing.Revision != nil && *existing.Revision >= next {
			return conflict("resource policy history already records this revision")
		}
	}
	return nil
}

func (hooks *PolicyRules) current(head *ResourcePolicyHead) *CurrentPolicyView {
	if hooks == nil || hooks.Input == nil {
		return nil
	}
	for _, candidate := range hooks.Input.CurrentPolicy {
		if candidate != nil && candidate.TenantId == head.TenantId && candidate.ResourceKind == head.ResourceKind && candidate.ResourceId == head.ResourceId && candidate.ResourceVersion == head.ResourceVersion {
			return candidate
		}
	}
	return nil
}

func (hooks *PolicyRules) history(head *ResourcePolicyHead) []*CurrentHistoryView {
	if hooks == nil || hooks.Input == nil {
		return nil
	}
	var result []*CurrentHistoryView
	for _, candidate := range hooks.Input.CurrentHistory {
		if candidate != nil && candidate.TenantId == head.TenantId && candidate.ResourceKind == head.ResourceKind && candidate.ResourceId == head.ResourceId && candidate.ResourceVersion == head.ResourceVersion {
			result = append(result, candidate)
		}
	}
	return result
}

func (hooks *PolicyRules) AfterSequence(context.Context, *ResourcePolicyHead, xhandler.LifecycleContext[ResourcePolicyHead, xhandler.NoParent, Output]) error {
	return nil
}

func (hooks *PolicyRules) AfterQueue(context.Context, *ResourcePolicyHead, xhandler.LifecycleContext[ResourcePolicyHead, xhandler.NoParent, Output]) error {
	return nil
}

func (hooks *PolicyRules) Finalize(context.Context, *Input, *Output, xhandler.Outcome) error {
	return nil
}
