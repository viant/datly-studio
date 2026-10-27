package host

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	sqliteDriver "modernc.org/sqlite"
)

var independentGateOnce sync.Once
var independentGateError error
var independentGates sync.Map

type independentQueryGate struct {
	mu      sync.Mutex
	calls   map[int64]int
	armed   bool
	arrived chan int64
	release chan struct{}
}

func (g *independentQueryGate) enter(value int64) (driver.Value, error) {
	g.mu.Lock()
	g.calls[value]++
	armed, arrived, release := g.armed, g.arrived, g.release
	g.mu.Unlock()
	if armed {
		arrived <- value
		select {
		case <-release:
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("independent query fixture barrier timed out")
		}
	}
	return int64(1), nil
}

func (g *independentQueryGate) snapshot() map[int64]int {
	g.mu.Lock()
	defer g.mu.Unlock()
	result := make(map[int64]int, len(g.calls))
	for key, value := range g.calls {
		result[key] = value
	}
	return result
}

const independentLimitReaderDQL = `#package('example.com/runtime/records')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#define($_ = $MaximumProject<int>(query/max_project).Required())
#define($_ = $Ceiling<?>(view/Ceiling).Cardinality('One').WithConnector('source').WithColumnType('ceiling_value','*int') /* SELECT MAX(id) AS ceiling_value FROM tasks WHERE project_id <= :MaximumProject */)
#define($_ = $Records<[]*Record>(output/view))
SELECT records.*, type(records,'Record')
FROM (SELECT t.id,t.name FROM tasks t WHERE t.id <= $Ceiling.CeilingValue ORDER BY t.id) records`

func TestManagedReaderResolvesIndependentNullableView(t *testing.T) {
	independentGateOnce.Do(func() {
		independentGateError = sqliteDriver.RegisterScalarFunction("studio_input_view_gate", 2, func(_ *sqliteDriver.FunctionContext, args []driver.Value) (driver.Value, error) {
			key, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("independent query fixture key is required")
			}
			entry, found := independentGates.Load(key)
			value, typed := args[1].(int64)
			if !found || !typed {
				return nil, fmt.Errorf("independent query fixture is unavailable")
			}
			return entry.(*independentQueryGate).enter(value)
		})
	})
	require.NoError(t, independentGateError)
	gate := &independentQueryGate{calls: map[int64]int{}}
	key := t.Name()
	independentGates.Store(key, gate)
	t.Cleanup(func() { independentGates.Delete(key) })
	dependencySQL := "SELECT MAX(id) + 0*studio_input_view_gate('" + key + "',:MaximumProject) AS ceiling_value FROM tasks WHERE project_id <= :MaximumProject"
	dql := strings.Replace(independentLimitReaderDQL, "SELECT MAX(id) AS ceiling_value FROM tasks WHERE project_id <= :MaximumProject", dependencySQL, 1)
	identity := newStockOAuthFixture(t)
	fixture, err := newScopedHostConfigured(t, map[string]string{"records": dql}, map[string]authz.Policy{"records": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}, func(config *ResourceAccessConfig) {
		config.Tenant = "team"
		config.Provider = identity.provider
	})
	require.NoError(t, err)
	gate.mu.Lock()
	gate.calls = map[int64]int{}
	gate.mu.Unlock()
	for _, tc := range []struct {
		maximum int
		names   []string
	}{
		{102, []string{"alpha-101", "beta-102"}},
		{103, []string{"alpha-101", "beta-102", "gamma-103"}},
		{100, []string{}},
		{102, []string{"alpha-101", "beta-102"}},
	} {
		t.Run(fmt.Sprint(tc.maximum), func(t *testing.T) {
			subject := "alice"
			if tc.maximum == 103 {
				subject = "bob"
			}
			actor := identity.token(t, identity.private, subject, []string{"reader"}, time.Now().Add(time.Minute))
			code, body := fixture.get(t, fmt.Sprintf("/records?max_project=%d", tc.maximum), actor, nil)
			require.Equal(t, 200, code, body)
			var output struct {
				Records []struct {
					Name string `json:"name"`
				} `json:"Records"`
			}
			require.NoError(t, json.Unmarshal([]byte(body), &output))
			names := make([]string, 0, len(output.Records))
			for _, row := range output.Records {
				names = append(names, row.Name)
			}
			require.Equal(t, tc.names, names, "computed nullable member must reach the root query")
			require.NotContains(t, strings.Join(names, ","), "delta-104")
		})
	}
	require.Equal(t, map[int64]int{102: 2, 103: 1, 100: 1}, gate.snapshot(), "one dependency query per call, including nullable empty MAX")
	for _, denied := range []struct {
		key    bool
		roles  []string
		status int
	}{{false, []string{"reader"}, 401}, {true, []string{"visitor"}, 403}} {
		key := identity.other
		if denied.key {
			key = identity.private
		}
		token := identity.token(t, key, "alice", denied.roles, time.Now().Add(time.Minute))
		status, _ := fixture.get(t, "/records?max_project=102", token, nil)
		require.Equal(t, denied.status, status)
	}
	require.Equal(t, map[int64]int{102: 2, 103: 1, 100: 1}, gate.snapshot(), "denied actors must not invoke the dependency")

	gate.mu.Lock()
	gate.calls = map[int64]int{}
	gate.armed = true
	gate.arrived = make(chan int64, 2)
	gate.release = make(chan struct{})
	gate.mu.Unlock()
	var release sync.Once
	unblock := func() { release.Do(func() { close(gate.release) }) }
	defer unblock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	type response struct {
		maximum, status int
		body            string
		err             error
	}
	responses := make(chan response, 2)
	for _, call := range []struct {
		maximum int
		subject string
	}{{102, "alice"}, {103, "bob"}} {
		token := identity.token(t, identity.private, call.subject, []string{"reader"}, time.Now().Add(time.Minute))
		go func(maximum int, credential string) {
			request, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/records?max_project=%d", fixture.httpAddr, maximum), nil)
			if err != nil {
				responses <- response{maximum: maximum, err: err}
				return
			}
			request.Header.Set("Authorization", "Bearer "+credential)
			result, err := http.DefaultClient.Do(request)
			if err != nil {
				responses <- response{maximum: maximum, err: err}
				return
			}
			defer result.Body.Close()
			body, err := io.ReadAll(result.Body)
			responses <- response{maximum: maximum, status: result.StatusCode, body: string(body), err: err}
		}(call.maximum, token)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-gate.arrived:
		case <-ctx.Done():
			t.Fatal("managed dependency calls did not overlap")
		}
	}
	unblock()
	for i := 0; i < 2; i++ {
		result := <-responses
		require.NoError(t, result.err)
		require.Equal(t, 200, result.status, result.body)
		want := []string{"alpha-101", "beta-102"}
		if result.maximum == 103 {
			want = append(want, "gamma-103")
		}
		var output struct {
			Records []struct {
				Name string `json:"name"`
			} `json:"Records"`
		}
		require.NoError(t, json.Unmarshal([]byte(result.body), &output))
		names := make([]string, 0, len(output.Records))
		for _, row := range output.Records {
			names = append(names, row.Name)
		}
		require.Equal(t, want, names)
	}
	require.Equal(t, map[int64]int{102: 1, 103: 1}, gate.snapshot(), "overlapping callers must each resolve exactly once")
}

func TestManagedReaderIndependentDependencyFailsClosed(t *testing.T) {
	identity := newStockOAuthFixture(t)
	configure := func(config *ResourceAccessConfig) { config.Tenant = "team"; config.Provider = identity.provider }
	policy := map[string]authz.Policy{"records": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}
	t.Run("missing connector", func(t *testing.T) {
		dql := strings.Replace(independentLimitReaderDQL, "WithConnector('source')", "WithConnector('unregistered')", 1)
		_, err := newScopedHostConfigured(t, map[string]string{"records": dql}, policy, configure)
		require.Error(t, err)
	})
	t.Run("one row cardinality", func(t *testing.T) {
		dql := strings.Replace(independentLimitReaderDQL, "MAX(id) AS ceiling_value", "id AS ceiling_value", 1)
		fixture, err := newScopedHostConfigured(t, map[string]string{"records": dql}, policy, configure)
		require.NoError(t, err)
		token := identity.token(t, identity.private, "alice", []string{"reader"}, time.Now().Add(time.Minute))
		code, body := fixture.get(t, "/records?max_project=102", token, nil)
		require.Equal(t, 500, code, body)
		require.NotContains(t, body, "alpha-101")
		require.NotContains(t, body, "beta-102")
	})
}

func TestManagedReaderWithoutIndependentViewsUnchanged(t *testing.T) {
	identity := newStockOAuthFixture(t)
	fixture, err := newScopedHostConfigured(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}, func(config *ResourceAccessConfig) { config.Tenant = "team"; config.Provider = identity.provider })
	require.NoError(t, err)
	token := identity.token(t, identity.private, "alice", []string{"reader"}, time.Now().Add(time.Minute))
	code, body := fixture.get(t, "/records", token, nil)
	require.Equal(t, 200, code, body)
	var output struct {
		Records []struct {
			Name string `json:"name"`
		} `json:"Records"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &output))
	names := make([]string, 0, len(output.Records))
	for _, row := range output.Records {
		names = append(names, row.Name)
	}
	require.Equal(t, []string{"alpha-101", "beta-102", "gamma-103", "delta-104"}, names)
}
