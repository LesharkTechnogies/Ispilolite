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
)

var (
	HTTPRequests = prometheus.NewCounterVec(prometheus.CounterOpts{Name:"ispilo_http_requests_total",Help:"HTTP requests by service, route, method, and status."},[]string{"service","route","method","status"})
	HTTPDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name:"ispilo_http_request_duration_seconds",Help:"HTTP request latency.",Buckets:[]float64{.01,.025,.05,.1,.2,.3,.5,1,2,5}},[]string{"service","route","method"})
	HTTPInFlight = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name:"ispilo_http_requests_in_flight",Help:"Requests currently being served."},[]string{"service"})
	BusinessEvents = prometheus.NewCounterVec(prometheus.CounterOpts{Name:"ispilo_business_events_total",Help:"Completed business operations."},[]string{"event","result"})
	SearchRequests = prometheus.NewCounterVec(prometheus.CounterOpts{Name:"ispilo_search_requests_total",Help:"Search requests by backend and degradation state."},[]string{"source","degraded"})
	GeospatialDuration = prometheus.NewHistogram(prometheus.HistogramOpts{Name:"ispilo_geospatial_query_duration_seconds",Help:"Latency of geospatial HTTP operations.",Buckets:[]float64{.01,.025,.05,.1,.2,.3,.5,1,2}})
	QueuePublished = prometheus.NewCounterVec(prometheus.CounterOpts{Name:"ispilo_rabbitmq_published_total",Help:"RabbitMQ messages published."},[]string{"exchange","routing_key","result"})
	QueueConsumed = prometheus.NewCounterVec(prometheus.CounterOpts{Name:"ispilo_rabbitmq_consumed_total",Help:"RabbitMQ deliveries consumed."},[]string{"queue","result"})
	QueueDepth = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name:"ispilo_rabbitmq_queue_depth",Help:"Ready messages in RabbitMQ queues."},[]string{"queue"})
	DBOpen = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name:"ispilo_db_open_connections",Help:"Open database connections."},[]string{"pool"})
	DBInUse = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name:"ispilo_db_in_use_connections",Help:"In-use database connections."},[]string{"pool"})
	DBIdle = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name:"ispilo_db_idle_connections",Help:"Idle database connections."},[]string{"pool"})
	DBWait = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name:"ispilo_db_wait_count",Help:"Cumulative database connection waits."},[]string{"pool"})
)

func init(){prometheus.MustRegister(HTTPRequests,HTTPDuration,HTTPInFlight,BusinessEvents,SearchRequests,GeospatialDuration,QueuePublished,QueueConsumed,QueueDepth,DBOpen,DBInUse,DBIdle,DBWait)}
func Handler()http.Handler{return promhttp.Handler()}

type statusRecorder struct{http.ResponseWriter;status int}
func (w *statusRecorder) WriteHeader(code int){w.status=code;w.ResponseWriter.WriteHeader(code)}
func (w *statusRecorder) Write(body []byte)(int,error){if w.status==0{w.WriteHeader(http.StatusOK)};return w.ResponseWriter.Write(body)}

func Middleware(service string,next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.URL.Path=="/metrics"{next.ServeHTTP(w,r);return};start:=time.Now();HTTPInFlight.WithLabelValues(service).Inc();defer HTTPInFlight.WithLabelValues(service).Dec();rec:=&statusRecorder{ResponseWriter:w};next.ServeHTTP(rec,r);status:=rec.status;if status==0{status=200};route:=normalizeRoute(r.URL.Path);duration:=time.Since(start).Seconds();HTTPRequests.WithLabelValues(service,route,r.Method,strconv.Itoa(status)).Inc();HTTPDuration.WithLabelValues(service,route,r.Method).Observe(duration);if strings.HasPrefix(route,"/api/v1/geo/"){GeospatialDuration.Observe(duration)}})}

func normalizeRoute(path string)string{parts:=strings.Split(strings.Trim(path,"/"),"/");for i,p:=range parts{if looksLikeID(p){parts[i]="{id}"}};if len(parts)==0{return "/"};return "/"+strings.Join(parts,"/")}
func looksLikeID(v string)bool{if len(v)>=24{return true};if strings.HasPrefix(strings.ToUpper(v),"ILO")&&len(v)>=8{return true};return false}

func ObserveDB(pool string,db *sql.DB){if db==nil{return};stats:=db.Stats();DBOpen.WithLabelValues(pool).Set(float64(stats.OpenConnections));DBInUse.WithLabelValues(pool).Set(float64(stats.InUse));DBIdle.WithLabelValues(pool).Set(float64(stats.Idle));DBWait.WithLabelValues(pool).Set(float64(stats.WaitCount))}

func StartDBCollector(stop <-chan struct{},writer,reader *sql.DB){ticker:=time.NewTicker(15*time.Second);go func(){defer ticker.Stop();for{select{case <-ticker.C:ObserveDB("writer",writer);ObserveDB("reader",reader);case <-stop:return}}}}()}
