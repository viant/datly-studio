package connectivity

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"syscall"

	"github.com/go-sql-driver/mysql"
)

// ProbeError exposes only a bounded, stage-specific diagnostic. The original
// driver/resolver error stays server-side because it may contain credentials.
type ProbeError struct {
	Code, Message string
	cause         error
}

func (e *ProbeError) Error() string { return e.Message }
func (e *ProbeError) Unwrap() error { return e.cause }

func probeFailure(stage string, cause error) error {
	code, message := "connectivity_failed", "Connector connectivity check failed."
	var mysqlError *mysql.MySQLError
	var dnsError *net.DNSError
	var networkError net.Error
	switch {
	case errors.Is(cause, context.DeadlineExceeded):
		code, message = "connection_timeout", "Connection test timed out. Check the database host, port, firewall, and network route."
	case errors.Is(cause, context.Canceled):
		code, message = "connection_canceled", "Connection test was canceled."
	case stage == "secret" && errors.Is(cause, fs.ErrNotExist):
		code, message = "secret_not_found", "Connector secret file was not found on this server. Check the secret URL and deployment volume mount."
	case stage == "secret" && errors.Is(cause, fs.ErrPermission):
		code, message = "secret_permission_denied", "The server cannot read the connector secret file. Check the file and volume permissions."
	case stage == "secret":
		code, message = "secret_resolution_failed", "Could not load or decrypt the connector secret. Check the secret URL, encryption key, and secret format."
	case errors.As(cause, &mysqlError):
		switch mysqlError.Number {
		case 1045:
			code, message = "authentication_failed", "MySQL rejected the database credentials (1045). Check the secret username/password and the account's allowed source hosts."
		case 1044:
			code, message = "database_access_denied", "MySQL denied access to the selected database (1044). Check the account's database grants."
		case 1049:
			code, message = "database_not_found", "MySQL could not find the selected database (1049). Check the database name in the DSN."
		case 1040:
			code, message = "connection_limit", "MySQL has reached its connection limit (1040). Try again after connections are released."
		default:
			code, message = "database_rejected_connection", "The database rejected the connection. Check the database server logs and connection settings."
		}
	case errors.As(cause, &dnsError):
		code, message = "dns_lookup_failed", "Could not resolve the database hostname. Check the DSN hostname and server DNS configuration."
	case errors.Is(cause, syscall.ECONNREFUSED):
		code, message = "connection_refused", "The database host refused the connection. Check the port and whether the database is listening."
	case errors.Is(cause, syscall.EHOSTUNREACH) || errors.Is(cause, syscall.ENETUNREACH):
		code, message = "network_unreachable", "The database network is unreachable from this server. Check routing and firewall rules."
	case errors.As(cause, &networkError) && networkError.Timeout():
		code, message = "connection_timeout", "Connection test timed out. Check the database host, port, firewall, and network route."
	case stage == "open":
		code, message = "invalid_connection_config", "The driver could not open the connection configuration. Check the DSN syntax and driver settings."
	case stage == "initialize":
		code, message = "connection_initialization_failed", "Connection initialization failed. Check the connector options and database permissions."
	}
	return &ProbeError{Code: code, Message: message, cause: cause}
}

// Diagnostic never forwards arbitrary driver error strings to clients.
func Diagnostic(err error) (code, message string) {
	var failure *ProbeError
	if errors.As(err, &failure) {
		return failure.Code, failure.Message
	}
	return "connectivity_failed", "Connector connectivity check failed."
}
