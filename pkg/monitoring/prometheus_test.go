package monitoring

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

var registerHealthDrivers sync.Once

func TestObservePostgresHealth(t *testing.T) {
	registerTestHealthDrivers()

	t.Run("records query results", func(t *testing.T) {
		db, err := sql.Open("postgres-health-success", "")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ObservePostgresHealth("test-success", db)

		assertMetric(t, `ispilo_postgres_up{pool="test-success"} 1`)
		assertMetric(t, `ispilo_postgres_waiting_locks{pool="test-success"} 2`)
		assertMetric(t, `ispilo_postgres_slow_queries{pool="test-success"} 3`)
		assertMetric(t, `ispilo_postgres_replication_lag_seconds{pool="test-success"} 4.5`)
	})

	t.Run("clears unavailable pool values", func(t *testing.T) {
		db, err := sql.Open("postgres-health-ping-error", "")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		setPostgresHealth("test-unavailable", true, 7, 8, 9)
		ObservePostgresHealth("test-unavailable", db)

		assertMetric(t, `ispilo_postgres_up{pool="test-unavailable"} 0`)
		assertMetric(t, `ispilo_postgres_waiting_locks{pool="test-unavailable"} 0`)
		assertMetric(t, `ispilo_postgres_slow_queries{pool="test-unavailable"} 0`)
		assertMetric(t, `ispilo_postgres_replication_lag_seconds{pool="test-unavailable"} 0`)
	})

	t.Run("uses zero when health queries fail", func(t *testing.T) {
		db, err := sql.Open("postgres-health-query-error", "")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ObservePostgresHealth("test-query-error", db)

		assertMetric(t, `ispilo_postgres_up{pool="test-query-error"} 1`)
		assertMetric(t, `ispilo_postgres_waiting_locks{pool="test-query-error"} 0`)
		assertMetric(t, `ispilo_postgres_slow_queries{pool="test-query-error"} 0`)
		assertMetric(t, `ispilo_postgres_replication_lag_seconds{pool="test-query-error"} 0`)
	})
}

func assertMetric(t *testing.T, want string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/metrics", nil)
	Handler().ServeHTTP(recorder, request)
	if !strings.Contains(recorder.Body.String(), want) {
		t.Fatalf("metric output does not contain %q", want)
	}
}

func registerTestHealthDrivers() {
	registerHealthDrivers.Do(func() {
		sql.Register("postgres-health-success", healthDriver{})
		sql.Register("postgres-health-ping-error", healthDriver{pingErr: errors.New("unavailable")})
		sql.Register("postgres-health-query-error", healthDriver{queryErr: errors.New("query failed")})
	})
}

type healthDriver struct {
	pingErr  error
	queryErr error
}

func (d healthDriver) Open(string) (driver.Conn, error) {
	return healthConn{pingErr: d.pingErr, queryErr: d.queryErr}, nil
}

type healthConn struct {
	pingErr  error
	queryErr error
}

func (c healthConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (c healthConn) Close() error                        { return nil }
func (c healthConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }
func (c healthConn) Ping(context.Context) error          { return c.pingErr }

func (c healthConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.queryErr != nil {
		return nil, c.queryErr
	}

	var value driver.Value
	switch {
	case strings.Contains(query, "wait_event_type"):
		value = int64(2)
	case strings.Contains(query, "query_start"):
		value = int64(3)
	case strings.Contains(query, "pg_last_xact_replay_timestamp"):
		value = 4.5
	default:
		return nil, errors.New("unexpected query")
	}
	return &healthRows{value: value}, nil
}

type healthRows struct {
	value driver.Value
	done  bool
}

func (r *healthRows) Columns() []string { return []string{"value"} }
func (r *healthRows) Close() error      { return nil }
func (r *healthRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	dest[0] = r.value
	r.done = true
	return nil
}
