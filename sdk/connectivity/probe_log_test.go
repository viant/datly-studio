package connectivity

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/sdk"
)

func TestProbeLogIdentifiesHostAndRevisionWithoutSecrets(t *testing.T) {
	var output bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	defer slog.SetDefault(original)
	connector := &sdk.Connector{Name: "example", Driver: "mysql", ETag: 17, DSNTemplate: "private-dsn", SecretRef: "private-secret"}
	err := probeFailure("ping", &mysql.MySQLError{Number: 1045, SQLState: [5]byte{'2', '8', '0', '0', '0'}, Message: "credential-private-payload"})
	logProbe(context.Background(), connector, "ping", probeDestination("mysql", "user:private-password@tcp(db.example:3306)/sample"), time.Now(), err)
	logged := output.String()
	for _, field := range []string{`"host":`, `"hostIPs":`, `"processID":`, `"connectorRevision":17`, `"mysqlCode":1045`, `"sqlState":"28000"`, `"status":"failed"`, `"destinationFingerprint":`} {
		if !strings.Contains(logged, field) {
			t.Errorf("missing field %s", field)
		}
	}
	for _, secret := range []string{"private-dsn", "private-secret", "private-password", "credential-private-payload", "db.example"} {
		if strings.Contains(logged, secret) {
			t.Errorf("sensitive value leaked")
		}
	}
}
func TestProbeDestinationIgnoresCredentialsButSeparatesTargets(t *testing.T) {
	first := probeDestination("mysql", "one:first@tcp(db.example:3306)/sample")
	if first == "" || first != probeDestination("mysql", "two:second@tcp(db.example:3306)/sample?parseTime=true") {
		t.Fatal("destination depends on credentials/options")
	}
	if first == probeDestination("mysql", "one:first@tcp(other.example:3306)/sample") {
		t.Fatal("distinct destination collapsed")
	}
	if probeDestination("mysql", "invalid") != "" {
		t.Fatal("invalid DSN fingerprinted")
	}
}
