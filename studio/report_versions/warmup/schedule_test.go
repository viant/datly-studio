package warmup_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/internal/datatest"
	studiocodecs "github.com/viant/datly-studio/runtime/codecs"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/host"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func testWarmupSchedules(ctx context.Context, t *testing.T, db *sql.DB, ns, bearer, address string, authoring *host.Authoring, jwt *datatest.JWTFixture, call func(context.Context, string, string, map[string]any) (json.RawMessage, bool)) {
	t.Helper()
	args := map[string]any{"namespaceId": ns, "reportId": "named-root", "versionNo": 1}
	first := time.Now().UTC().Truncate(time.Microsecond).Add(-3 * time.Hour)
	args["input"] = sdk.SaveWarmupScheduleInput{IntervalSeconds: 3600, FirstRunAt: first, Enabled: true}
	raw, ok := call(ctx, "studio.sdk.versions.warmup_schedule_save", bearer, args)
	var result sdk.WarmupScheduleResult
	if !ok || json.Unmarshal(raw, &result) != nil || result.Schedule == nil || result.Schedule.Revision != 1 {
		t.Fatalf("native schedule save: %s", raw)
	}
	if _, ok = call(ctx, "studio.sdk.versions.warmup_schedule_save", bearer, args); ok {
		t.Fatal("native save accepted stale revision")
	}
	viewer := jwt.BearerWithClaims(t, jwtv5.MapClaims{"sub": "viewer", "iss": "warmup-test", "aud": "studio", "exp": time.Now().Add(time.Hour).Unix()})
	if _, ok = call(ctx, "studio.sdk.versions.warmup_schedule_get", viewer, args); ok {
		t.Fatal("inspection-only caller read schedule")
	}
	// Use the real native HTTP read path too.
	body, _ := json.Marshal(map[string]any{"reportId": "named-root", "versionNo": 1})
	req, e := http.NewRequestWithContext(ctx, "POST", "http://"+address+"/v1/studio/sdk/versions.warmup_schedule_get", bytes.NewReader(body))
	if e != nil {
		t.Fatal(e)
	}
	req.Header.Set("Authorization", bearer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Studio-Namespace", ns)
	response, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	data, e := io.ReadAll(response.Body)
	response.Body.Close()
	if e != nil || response.StatusCode != 200 || !bytes.Contains(data, []byte(result.Schedule.ScheduleID)) {
		t.Fatalf("native HTTP schedule: %s %v", data, e)
	}
	service := sdk.SystemPrincipal().Subject
	if _, e = db.ExecContext(ctx, "INSERT INTO component_acl(namespace_id,report_id,subject_type,subject_id,can_view,can_publish) VALUES(?,?,'user',?,1,1)", ns, "named-root", service); e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, "UPDATE connectors SET owner_id=? WHERE name='batch'", service); e != nil {
		t.Fatal(e)
	}
	claims := jwtv5.MapClaims{"sub": service, "user_id": 1, "iss": "warmup-test", "aud": "studio", "tenant": "tenant", "exp": time.Now().Add(time.Hour).Unix()}
	token := jwt.BearerWithClaims(t, claims)
	credentials := func(context.Context) (sdk.VerifiedCredential, error) {
		return sdk.VerifiedCredential{Bearer: token, Claims: jwtv5.RegisteredClaims{Subject: service, Issuer: "warmup-test", Audience: jwtv5.ClaimStrings{"studio"}, ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour))}}, nil
	}
	config := host.AuthoringConfig{Config: host.Config{}, ModulePath: authoring.ModulePath(), ReaderCaches: authoring.ReaderCaches(), SourceValidator: authoring.SourceValidator(), Dependencies: authoring.Dependencies()}
	timeout := "10m"
	config.Config.WarmupOperationTimeout = &timeout
	runLoop := func(observe func(error)) (context.CancelFunc, <-chan error) {
		c, stop := context.WithCancel(ctx)
		done := make(chan error, 1)
		go func() {
			done <- host.RunWarmupScheduler(c, db, config, credentials, studiocodecs.New(jwt.Factory), observe)
		}()
		return stop, done
	}
	var before string
	if e = db.QueryRowContext(ctx, "SELECT authored_dql FROM component_versions WHERE report_id='named-root' AND version_no=1").Scan(&before); e != nil {
		t.Fatal(e)
	}
	stop, done := runLoop(func(e error) { t.Logf("admission: %v", e) })
	defer stop()
	until := time.Now().Add(8 * time.Second)
	var runID, status, actor string
	for time.Now().Before(until) {
		e = db.QueryRowContext(ctx, "SELECT r.run_id,r.status,r.requested_by FROM component_warmup_schedules s JOIN component_warmup_runs r ON r.run_id=s.last_run_id WHERE s.report_id='named-root'").Scan(&runID, &status, &actor)
		if e == nil && (status == "completed" || status == "failed" || status == "partial") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stop()
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if status != "completed" || actor != service {
		var diagnostic string
		_ = db.QueryRowContext(ctx, "SELECT diagnostics_json FROM component_warmup_runs WHERE run_id=?", runID).Scan(&diagnostic)
		t.Fatalf("service run=%s status=%s actor=%s err=%v diagnostics=%s", runID, status, actor, e, diagnostic)
	}
	raw, ok = call(ctx, "studio.sdk.versions.warmup_schedule_get", bearer, args)
	if !ok || json.Unmarshal(raw, &result) != nil || result.Schedule.Revision != 2 || result.Schedule.LastRunID == nil || *result.Schedule.LastRunID != runID || !result.Schedule.NextDueAt.After(time.Now()) {
		t.Fatalf("scheduled readback %s", raw)
	}
	var after string
	if e = db.QueryRowContext(ctx, "SELECT authored_dql FROM component_versions WHERE report_id='named-root' AND version_no=1").Scan(&after); e != nil || after != before {
		t.Fatal("scheduling changed authored source")
	}
	// A fresh due occurrence must fail admission once the service loses publish.
	args["input"] = sdk.SaveWarmupScheduleInput{ExpectedRevision: 2, IntervalSeconds: 3600, FirstRunAt: time.Now().UTC().Add(-time.Minute), Enabled: true}
	if raw, ok = call(ctx, "studio.sdk.versions.warmup_schedule_save", bearer, args); !ok {
		t.Fatalf("reschedule %s", raw)
	}
	if _, e = db.ExecContext(ctx, "UPDATE component_acl SET can_publish=0 WHERE report_id='named-root' AND subject_id=?", service); e != nil {
		t.Fatal(e)
	}
	errors := make(chan error, 1)
	stop, done = runLoop(func(e error) {
		select {
		case errors <- e:
		default:
		}
	})
	select {
	case <-errors:
	case <-time.After(4 * time.Second):
		stop()
		<-done
		t.Fatal("service revocation was not observed")
	}
	stop()
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	var count int
	if e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_warmup_runs WHERE requested_by=?", service).Scan(&count); e != nil || count != 1 {
		t.Fatalf("revoked scheduler inserted work: %d %v", count, e)
	}
	raw, ok = call(ctx, "studio.sdk.versions.warmup_schedule_get", bearer, args)
	if !ok || json.Unmarshal(raw, &result) != nil || result.Schedule.Revision != 3 || !result.Schedule.NextDueAt.Before(time.Now()) {
		t.Fatalf("denial consumed occurrence %s", raw)
	}
	args["input"] = sdk.DeleteWarmupScheduleInput{ExpectedRevision: 3}
	if raw, ok = call(ctx, "studio.sdk.versions.warmup_schedule_delete", bearer, args); !ok {
		t.Fatalf("native delete %s", raw)
	}
	raw, ok = call(ctx, "studio.sdk.versions.warmup_schedule_get", bearer, args)
	if !ok || strings.Contains(string(raw), "scheduleId") {
		t.Fatalf("schedule remains %s", raw)
	}
}
