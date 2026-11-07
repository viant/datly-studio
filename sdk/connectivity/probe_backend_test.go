package connectivity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/sdk"
)

func TestSQLProbeMySQLBackendIdentity(t *testing.T) {
	for _, scenario := range []string{"identity", "identity denied", "identity timeout", "ping denied"} {
		t.Run(scenario, func(t *testing.T) {
			var output bytes.Buffer
			original := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
			t.Cleanup(func() { slog.SetDefault(original) })

			// Exercise the linked MySQL driver and the public probe without a
			// database service. The server requires ping, identity query, and
			// close to occur in order on a single authenticated session.
			const network = "probe_identity_fixture"
			var dialCount atomic.Int32
			serverDone := make(chan error, 1)
			mysql.RegisterDialContext(network, func(_ context.Context, _ string) (net.Conn, error) {
				if dialCount.Add(1) != 1 {
					return nil, errors.New("unexpected second physical connection")
				}
				client, server := net.Pipe()
				go func() {
					defer server.Close()
					_ = server.SetDeadline(time.Now().Add(6 * time.Second))
					serverDone <- serveProbeIdentity(server, scenario)
				}()
				return client, nil
			})
			t.Cleanup(func() { mysql.DeregisterDialContext(network) })

			dsn := "fixture:expanded-secret@" + network + "(route.example)/selected_schema"
			probe := SQLProbe{Secrets: connectorsecret.ResolverFunc(func(_ context.Context, _, _ string) (string, error) {
				return dsn, nil
			})}
			result, err := probe.Probe(t.Context(), &sdk.Connector{
				Name: "probe fixture", Driver: "mysql", ETag: 17, DSNTemplate: "unexpanded-template", SecretRef: "fixture-secret-reference",
			})
			if dialCount.Load() != 1 {
				t.Fatalf("physical connections = %d; want 1", dialCount.Load())
			}
			select {
			case serverErr := <-serverDone:
				if serverErr != nil {
					t.Fatal(serverErr)
				}
			case <-time.After(6 * time.Second):
				t.Fatal("probe did not close the MySQL session")
			}

			wantStatus := "passed"
			if scenario == "ping denied" {
				wantStatus = "failed"
				if err == nil || result != nil {
					t.Fatalf("ping failure: result = %+v, err = %v", result, err)
				}
				if code, message := Diagnostic(err); code != "authentication_failed" || strings.Contains(message, "driver-private-payload") {
					t.Fatal("ping failure did not retain its safe authentication diagnostic")
				}
			} else if err != nil || result == nil || result.Status != wantStatus {
				t.Fatalf("result = %+v, err = %v; want passed", result, err)
			}

			var log map[string]any
			if err := json.Unmarshal(output.Bytes(), &log); err != nil {
				t.Fatalf("invalid probe log: %v", err)
			}
			for key, want := range map[string]any{
				"stage": "ping", "status": wantStatus, "connectorRevision": float64(17),
				"expandedDSNFingerprint": expandedProbeFingerprint(dsn), "destinationFingerprint": probeDestination("mysql", dsn),
			} {
				if log[key] != want {
					t.Errorf("%s = %v; want %v", key, log[key], want)
				}
			}
			for _, key := range []string{"host", "hostIPs", "processID"} {
				if _, ok := log[key]; !ok {
					t.Errorf("missing %s", key)
				}
			}
			for _, secret := range []string{"expanded-secret", "route.example", "unexpanded-template", "fixture-secret-reference", "driver-private-payload"} {
				if strings.Contains(output.String(), secret) {
					t.Error("sensitive value leaked in probe log")
				}
			}

			switch scenario {
			case "identity":
				for key, want := range map[string]any{
					"databaseHost": "backend.example", "databaseServerID": float64(7), "databaseVersion": "8.4.0", "databaseName": "selected_schema",
					"backendIdentityErrorType": nil,
				} {
					if log[key] != want {
						t.Errorf("%s = %v; want %v", key, log[key], want)
					}
				}
			default:
				for _, key := range []string{"databaseHost", "databaseServerID", "databaseVersion", "databaseName"} {
					if _, ok := log[key]; ok {
						t.Errorf("failed diagnostic logged %s", key)
					}
				}
				switch scenario {
				case "identity denied":
					if log["backendIdentityErrorType"] != "*mysql.MySQLError" {
						t.Error("diagnostic denial should log only the error type")
					}
				case "identity timeout":
					if log["backendIdentityErrorType"] != fmt.Sprintf("%T", context.DeadlineExceeded) {
						t.Error("diagnostic should time out before the connection deadline")
					}
				case "ping denied":
					if log["backendIdentityErrorType"] != nil || log["mysqlCode"] != float64(1045) {
						t.Error("ping denial should report the MySQL code and omit the diagnostic")
					}
				}
			}
		})
	}
}

// serveProbeIdentity speaks the MySQL handshake, ping, and a single text
// result set (or error). The packet script also detects extra commands or a
// query issued before the successful ping.
func serveProbeIdentity(conn net.Conn, scenario string) error {
	read := func() ([]byte, error) {
		var header [4]byte
		if _, err := io.ReadFull(conn, header[:]); err != nil {
			return nil, err
		}
		size := int(header[0]) | int(header[1])<<8 | int(header[2])<<16
		packet := make([]byte, size)
		_, err := io.ReadFull(conn, packet)
		return packet, err
	}
	write := func(seq byte, packet []byte) error {
		header := []byte{byte(len(packet)), 0, 0, seq}
		_, err := conn.Write(append(header, packet...))
		return err
	}
	expect := func(want string) error {
		packet, err := read()
		if err != nil {
			return err
		}
		if string(packet) != want {
			return fmt.Errorf("unexpected MySQL command: want command %d", want[0])
		}
		return nil
	}

	// Protocol 10, protocol-41 and secure-connection capability flags.
	handshake := "\x0a8.4.0\x00\x01\x00\x00\x00abcdefgh\x00\x09\x82\x21\x02\x00\x08\x00\x15" +
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00ijklmnopqrst\x00mysql_native_password\x00"
	ok := []byte{0, 0, 0, 2, 0, 0, 0}
	denied := append([]byte{0xff, 0x15, 0x04}, []byte("#28000driver-private-payload")...)
	if err := write(0, []byte(handshake)); err != nil {
		return err
	}
	if _, err := read(); err != nil {
		return err
	}
	if err := write(2, ok); err != nil {
		return err
	}
	if err := expect("\x0e"); err != nil {
		return err
	}
	if scenario == "ping denied" {
		if err := write(1, denied); err != nil {
			return err
		}
		return expect("\x01")
	}
	if err := write(1, ok); err != nil {
		return err
	}
	if err := expect("\x03SELECT @@hostname, @@server_id, VERSION(), DATABASE()"); err != nil {
		return err
	}

	switch scenario {
	case "identity denied":
		if err := write(1, denied); err != nil {
			return err
		}
	case "identity timeout":
		if _, err := read(); !errors.Is(err, io.EOF) {
			return fmt.Errorf("expected driver to close the timed-out session: %v", err)
		}
		return nil
	default:
		eof := []byte{0xfe, 0, 0, 2, 0}
		if err := write(1, []byte{4}); err != nil {
			return err
		}
		var row []byte
		for i, value := range []string{"backend.example", "7", "8.4.0", "selected_schema"} {
			column := []byte("\x03def\x00\x00\x00\x01v\x00\x0c\x21\x00\xff\x00\x00\x00\xfd\x00\x00\x00\x00\x00")
			if err := write(byte(i+2), column); err != nil {
				return err
			}
			row = append(append(row, byte(len(value))), value...)
		}
		for i, packet := range [][]byte{eof, row, eof} {
			if err := write(byte(i+6), packet); err != nil {
				return err
			}
		}
	}
	return expect("\x01") // COM_QUIT
}
