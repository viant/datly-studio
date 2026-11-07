package connectivity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/sdk"
)

// Random process-local key makes expanded connection fingerprints comparable
// within this process without publishing a reusable credential digest.
var probeFingerprintKey = func() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil
	}
	return key
}()

func expandedProbeFingerprint(dsn string) string {
	if len(probeFingerprintKey) == 0 || dsn == "" {
		return ""
	}
	mac := hmac.New(sha256.New, probeFingerprintKey)
	_, _ = mac.Write([]byte(dsn))
	return hex.EncodeToString(mac.Sum(nil))
}

// A destination digest excludes credentials, DSN parameters and secret references.
func probeDestination(driver, dsn string) string {
	if driver != "mysql" {
		return ""
	}
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256([]byte(config.Net + "\x00" + config.Addr + "\x00" + config.DBName))
	return hex.EncodeToString(digest[:])
}

func probeHost() (string, []string) {
	hostname, _ := os.Hostname()
	addresses, _ := net.InterfaceAddrs()
	ips := []string{}
	for _, address := range addresses {
		ip, _, err := net.ParseCIDR(address.String())
		if err == nil && !ip.IsLoopback() && !ip.IsUnspecified() {
			ips = append(ips, ip.String())
		}
	}
	sort.Strings(ips)
	return hostname, ips
}

func logProbe(ctx context.Context, connector *sdk.Connector, stage, destination string, started time.Time, err error, details ...probeDetails) {
	if connector == nil {
		return
	}
	host, ips := probeHost()
	status, code := "passed", ""
	if err != nil {
		status = "failed"
		code, _ = Diagnostic(err)
	}
	fields := []any{"connector", connector.Name, "connectorRevision", connector.ETag,
		"driver", connector.Driver, "stage", stage, "status", status, "errorCode", code,
		"host", host, "hostIPs", ips, "processID", os.Getpid(), "elapsedMs", time.Since(started).Milliseconds()}
	if destination != "" {
		fields = append(fields, "destinationFingerprint", destination)
	}
	if len(details) > 0 {
		detail := details[0]
		if detail.ExpandedDSNFingerprint != "" {
			fields = append(fields, "expandedDSNFingerprint", detail.ExpandedDSNFingerprint)
		}
		if detail.Backend != nil && err == nil {
			backend := detail.Backend
			if backend.ErrorType != "" {
				fields = append(fields, "backendIdentityErrorType", backend.ErrorType)
			} else {
				fields = append(fields, "databaseHost", backend.Host, "databaseServerID", backend.ServerID, "databaseVersion", backend.Version, "databaseName", backend.Database.String)
			}
		}
	}
	var dbErr *mysql.MySQLError
	if errors.As(err, &dbErr) {
		fields = append(fields, "mysqlCode", dbErr.Number, "sqlState", strings.TrimRight(string(dbErr.SQLState[:]), "\x00"))
	}
	if err != nil {
		slog.ErrorContext(ctx, "Connector connectivity probe", fields...)
	} else {
		slog.InfoContext(ctx, "Connector connectivity probe", fields...)
	}
}

type probeDetails struct {
	ExpandedDSNFingerprint string
	Backend                *mysqlProbeBackend
}
