package globalpredicate

import (
	"context"
	"strings"
	"testing"
)

type subject string

func (value subject) GlobalPublishSubject() string { return string(value) }

func TestGlobalPublishRequiresBoundSubject(t *testing.T) {
	if _, err := (&GlobalPublish{}).Compute(context.Background(), nil); err == nil {
		t.Fatal("missing bound input accepted")
	}
	if _, err := (&GlobalPublish{Input: subject("")}).Compute(context.Background(), nil); err == nil {
		t.Fatal("empty subject accepted")
	}
	criteria, err := (&GlobalPublish{Input: subject("alice")}).Compute(context.Background(), nil)
	if err != nil || len(criteria.Placeholders) != 2 || criteria.Placeholders[0] != "alice" || criteria.Placeholders[1] != "alice" ||
		!strings.Contains(criteria.Expression, "can_publish") {
		t.Fatalf("criteria=%+v err=%v", criteria, err)
	}
}
