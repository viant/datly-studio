package connectivity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestProbeDiagnosticsClassifyErrorsWithoutExposingDriverOrSecretData(t *testing.T) {
	const sensitive = "private-user:private-password@tcp(private-host)/db?token=private-token"
	cases := []struct {
		name, stage, code string
		err               error
	}{
		{"missing secret", "secret", "secret_not_found", &os.PathError{Op: "open", Path: sensitive, Err: os.ErrNotExist}},
		{"secret permissions", "secret", "secret_permission_denied", &os.PathError{Op: "open", Path: sensitive, Err: os.ErrPermission}},
		{"decrypt secret", "secret", "secret_resolution_failed", errors.New(sensitive)},
		{"authentication", "ping", "authentication_failed", &mysql.MySQLError{Number: 1045, Message: sensitive}},
		{"database grants", "ping", "database_access_denied", &mysql.MySQLError{Number: 1044, Message: sensitive}},
		{"database missing", "ping", "database_not_found", &mysql.MySQLError{Number: 1049, Message: sensitive}},
		{"DNS", "ping", "dns_lookup_failed", &net.DNSError{Err: sensitive, Name: sensitive}},
		{"refused", "ping", "connection_refused", fmt.Errorf("%s: %w", sensitive, syscall.ECONNREFUSED)},
		{"timeout", "ping", "connection_timeout", context.DeadlineExceeded},
		{"invalid DSN", "open", "invalid_connection_config", errors.New(sensitive)},
		{"unknown driver error", "ping", "connectivity_failed", errors.New(sensitive)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := probeFailure(tc.stage, tc.err)
			code, message := Diagnostic(fmt.Errorf("wrapped: %w", wrapped))
			if code != tc.code || message == "" {
				t.Fatalf("diagnostic=%q %q", code, message)
			}
			for _, secret := range []string{"private-user", "private-password", "private-host", "private-token"} {
				if strings.Contains(message, secret) || strings.Contains(wrapped.Error(), secret) {
					t.Fatal("diagnostic leaked sensitive error data")
				}
			}
		})
	}
	code, message := Diagnostic(errors.New(sensitive))
	if code != "connectivity_failed" || strings.Contains(message, "private-") {
		t.Fatal("untrusted probe error was forwarded")
	}
}
