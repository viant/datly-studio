package connectivity

import (
	"context"
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

func logProbe(ctx context.Context, connector *sdk.Connector, stage, destination string, started time.Time, err error) {
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
