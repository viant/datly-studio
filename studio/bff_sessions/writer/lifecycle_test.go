package writer

import (
	"context"
	"testing"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
)

func TestSessionRevocationRulesRejectCrossSubjectAndMutation(t *testing.T) {
	zero := 0
	rules := &SessionRevocationRules{Input: &Input{
		Jwt:            &jwt.Claims{RegisteredClaims: registeredClaims("owner")},
		CurrentSession: []*CurrentSessionView{{SessionIdHash: "hash", SubjectId: "owner", ExpiresAtUnix: intPtr(100)}},
	}}
	state := xhandler.LifecycleContext[SessionRevocation, xhandler.NoParent, Output]{}
	if err := rules.Validate(context.Background(), &SessionRevocation{SessionIdHash: "hash", SubjectId: "owner", ExpiresAtUnix: &zero}, state); err != nil {
		t.Fatalf("own revocation rejected: %v", err)
	}
	for name, value := range map[string]*SessionRevocation{
		"cross subject": {SessionIdHash: "hash", SubjectId: "attacker", ExpiresAtUnix: &zero},
		"extend":        {SessionIdHash: "hash", SubjectId: "owner", ExpiresAtUnix: intPtr(200)},
		"unknown":       {SessionIdHash: "unknown", SubjectId: "owner", ExpiresAtUnix: &zero},
	} {
		t.Run(name, func(t *testing.T) {
			if err := rules.Validate(context.Background(), value, state); err == nil {
				t.Fatal("unsafe session mutation was accepted")
			}
		})
	}
}

func registeredClaims(subject string) jwtv5.RegisteredClaims {
	return jwtv5.RegisteredClaims{Subject: subject}
}

func intPtr(value int) *int { return &value }
