package monitoring

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var (
	HTTPRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ispilo_http_requests_total",
		Help: "HTTP requests by service, route, method, and status.",
	}, []string{"service", "route", "method", "status"})

	HTTPDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ispilo_http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: []float64{.01, .025, .05, .1, .2, .3, .5, 1, 2, 5},
	}, []string{"service", "route", "method"})

	HTTPInFlight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_http_requests_in_flight",
		Help: "Requests currently being served.",
	}, []string{"service"})

	BusinessEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ispilo_business_events_total",
		Help: "Completed business operations.",
	}, []string{"event", "result"})

	SearchRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ispilo_search_requests_total",
		Help: "Search requests by backend and degradation state.",
	}, []string{"source", "degraded"})

	GeospatialDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "ispilo_geospatial_query_duration_seconds",
		Help:    "Latency of geospatial HTTP operations.",
		Buckets: []float64{.01, .025, .05, .1, .2, .3, .5, 1, 2},
	})

	QueuePublished = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ispilo_rabbitmq_published_total",
		Help: "RabbitMQ messages published.",
	}, []string{"exchange", "routing_key", "result"})

	QueueConsumed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ispilo_rabbitmq_consumed_total",
		Help: "RabbitMQ deliveries consumed.",
	}, []string{"queue", "result"})

	QueueDepth = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_rabbitmq_queue_depth",
		Help: "Ready messages in RabbitMQ queues.",
	}, []string{"queue"})

	DBOpen = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_db_open_connections",
		Help: "Open database connections.",
	}, []string{"pool"})

	DBInUse = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_db_in_use_connections",
		Help: "In-use database connections.",
	}, []string{"pool"})

	DBIdle = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_db_idle_connections",
		Help: "Idle database connections.",
	}, []string{"pool"})

	DBWait = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_db_wait_count",
		Help: "Cumulative database connection waits.",
	}, []string{"pool"})

	PostgresUp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_postgres_up",
		Help: "Whether the PostgreSQL pool is reachable.",
	}, []string{"pool"})

	PostgresWaitingLocks = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_postgres_waiting_locks",
		Help: "Current PostgreSQL sessions waiting on locks.",
	}, []string{"pool"})

	PostgresSlowQueries = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_postgres_slow_queries",
		Help: "Current PostgreSQL queries running longer than five minutes.",
	}, []string{"pool"})

	PostgresReplicaLag = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ispilo_postgres_replication_lag_seconds",
		Help: "PostgreSQL replica replay lag in seconds.",
	}, []string{"pool"})
)

func init() {
	prometheus.MustRegister(
		HTTPRequests,
		HTTPDuration,
		HTTPInFlight,
		BusinessEvents,
		SearchRequests,
		GeospatialDuration,
		QueuePublished,
		QueueConsumed,
		QueueDepth,
		DBOpen,
		DBInUse,
		DBIdle,
		DBWait,
		PostgresUp,
		PostgresWaitingLocks,
		PostgresSlowQueries,
		PostgresReplicaLag,
	)
}

func Handler() http.Handler {
	return promhttp.Handler()
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func Middleware(service string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		parentCtx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := otel.Tracer("ispilolite/http").Start(parentCtx, "http "+r.Method+" "+r.URL.Path, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		r = r.WithContext(ctx)
		start := time.Now()
		HTTPInFlight.WithLabelValues(service).Inc()
		defer HTTPInFlight.WithLabelValues(service).Dec()

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = 200
		}

		route := normalizeRoute(r.URL.Path)
		duration := time.Since(start).Seconds()

		HTTPRequests.WithLabelValues(service, route, r.Method, strconv.Itoa(status)).Inc()
		HTTPDuration.WithLabelValues(service, route, r.Method).Observe(duration)

		if strings.HasPrefix(route, "/api/v1/geo/") {
			GeospatialDuration.Observe(duration)
		}
	})
}

func normalizeRoute(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if looksLikeID(p) {
			parts[i] = "{id}"
		}
	}
	if len(parts) == 0 {
		return "/"
	}
	return "/" + strings.Join(parts, "/")
}

func looksLikeID(v string) bool {
	if len(v) >= 24 {
		return true
	}
	if strings.HasPrefix(strings.ToUpper(v), "ILO") && len(v) >= 8 {
		return true
	}
	return false
}

func ObserveDB(pool string, db *sql.DB) {
	if db == nil {
		return
	}
	stats := db.Stats()
	DBOpen.WithLabelValues(pool).Set(float64(stats.OpenConnections))
	DBInUse.WithLabelValues(pool).Set(float64(stats.InUse))
	DBIdle.WithLabelValues(pool).Set(float64(stats.Idle))
	DBWait.WithLabelValues(pool).Set(float64(stats.WaitCount))
}

func StartDBCollector(stop <-chan struct{}, writer, reader *sql.DB) {
	ObserveDB("writer", writer)
	ObserveDB("reader", reader)
	ObservePostgresHealth("writer", writer)
	ObservePostgresHealth("reader", reader)

	ticker := time.NewTicker(15 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ObserveDB("writer", writer)
				ObserveDB("reader", reader)
				ObservePostgresHealth("writer", writer)
				ObservePostgresHealth("reader", reader)
			case <-stop:
				return
			}
		}
	}()
}

const postgresHealthTimeout = 5 * time.Second

// ObservePostgresHealth samples the PostgreSQL views used by the production
// dashboards. Query failures are represented as zero for that scrape; the up
// gauge distinguishes an unavailable pool from a healthy pool with no issues.
func ObservePostgresHealth(pool string, db *sql.DB) {
	if db == nil {
		setPostgresHealth(pool, false, 0, 0, 0)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), postgresHealthTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		setPostgresHealth(pool, false, 0, 0, 0)
		return
	}

	var waitingLocks, slowQueries int
	var replicaLag float64
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock' AND state <> 'idle'`).Scan(&waitingLocks); err != nil {
		waitingLocks = 0
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity WHERE state <> 'idle' AND query_start IS NOT NULL AND now() - query_start > interval '5 minutes'`).Scan(&slowQueries); err != nil {
		slowQueries = 0
	}
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())), 0)`).Scan(&replicaLag); err != nil {
		replicaLag = 0
	}
	if replicaLag < 0 {
		replicaLag = 0
	}
	setPostgresHealth(pool, true, float64(waitingLocks), float64(slowQueries), replicaLag)
}

func setPostgresHealth(pool string, up bool, waitingLocks, slowQueries, replicaLag float64) {
	if up {
		PostgresUp.WithLabelValues(pool).Set(1)
	} else {
		PostgresUp.WithLabelValues(pool).Set(0)
	}
	PostgresWaitingLocks.WithLabelValues(pool).Set(waitingLocks)
	PostgresSlowQueries.WithLabelValues(pool).Set(slowQueries)
	PostgresReplicaLag.WithLabelValues(pool).Set(replicaLag)
}
