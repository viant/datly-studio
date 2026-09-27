package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLinkSyncIsExplicitAndDefaultsToDependencyLink(t *testing.T) {
	for _, check := range []struct {
		args []string
		code int
	}{
		{[]string{"link"}, 2},
		{[]string{"link", "other"}, 2},
		{[]string{"link", "sync", "-h"}, 0},
		{[]string{"link", "sync", "-dir", t.TempDir()}, 1},
	} {
		var output, diagnostic bytes.Buffer
		if code := runLink(context.Background(), check.args, &output, &diagnostic); code != check.code {
			t.Fatalf("%v: exit=%d want=%d diagnostic=%s", check.args, code, check.code, diagnostic.String())
		}
		if check.code == 0 && !strings.Contains(diagnostic.String(), "internal/dependencylink") {
			t.Fatalf("link sync help omits default: %s", diagnostic.String())
		}
	}
}
