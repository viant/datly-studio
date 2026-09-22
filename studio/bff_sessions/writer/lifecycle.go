package writer

import (
	context "context"
	"fmt"
	xhandler2 "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// SessionRevocationRules customizes role Input.Sessions.
type SessionRevocationRules struct {
	Input *Input `bind:"kind=input,required"`
}

func SessionRevocationRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*SessionRevocationRules)(nil)).Elem()
}

var (
	SessionRevocationRulesHooks = new(SessionRevocationRules)
	SessionRevocationRulesDatly = SessionRevocationRulesDatlyType()
)

func (hooks *SessionRevocationRules) Init(ctx context.Context, entity *SessionRevocation, state xhandler2.LifecycleContext[SessionRevocation, xhandler2.NoParent, Output]) error {
	return nil
}
func (hooks *SessionRevocationRules) Validate(ctx context.Context, entity *SessionRevocation, state xhandler2.LifecycleContext[SessionRevocation, xhandler2.NoParent, Output]) error {
	if hooks == nil || hooks.Input == nil || hooks.Input.Jwt == nil {
		return fmt.Errorf("verified JWT input is required")
	}
	subject := strings.TrimSpace(hooks.Input.Jwt.Subject)
	if subject == "" || entity == nil || entity.SubjectId != subject {
		return fmt.Errorf("session revocation is limited to the verified JWT subject")
	}
	if entity.ExpiresAtUnix == nil || *entity.ExpiresAtUnix != 0 {
		return fmt.Errorf("session revocation may only set expiration to zero")
	}
	for _, current := range hooks.Input.CurrentSession {
		if current != nil && current.SessionIdHash == entity.SessionIdHash {
			if current.SubjectId != subject {
				return fmt.Errorf("session does not belong to the verified JWT subject")
			}
			return nil
		}
	}
	return fmt.Errorf("session does not exist")
}
func (hooks *SessionRevocationRules) AfterSequence(ctx context.Context, entity *SessionRevocation, state xhandler2.LifecycleContext[SessionRevocation, xhandler2.NoParent, Output]) error {
	return nil
}
func (hooks *SessionRevocationRules) AfterQueue(ctx context.Context, entity *SessionRevocation, state xhandler2.LifecycleContext[SessionRevocation, xhandler2.NoParent, Output]) error {
	return nil
}
func (hooks *SessionRevocationRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler2.Outcome) error {
	return nil
}
