package middleware

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net/http"
	"os"
	"time"
)

type requestIDKey struct{}
type logRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *logRecorder) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (w *logRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	n, err := w.ResponseWriter.Write(body)
	w.bytes += n
	return n, err
}
func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}
func RequestLogger(logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.New(os.Stdout, "", 0)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = uuid.NewString()
			}
			w.Header().Set("X-Request-ID", id)
			ctx := context.WithValue(r.Context(), requestIDKey{}, id)
			start := time.Now()
			rec := &logRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r.WithContext(ctx))
			status := rec.status
			if status == 0 {
				status = 200
			}
			event := map[string]interface{}{"timestamp": time.Now().UTC().Format(time.RFC3339Nano), "level": "info", "message": "http_request", "request_id": id, "method": r.Method, "path": r.URL.Path, "query": r.URL.RawQuery, "status": status, "response_bytes": rec.bytes, "duration_ms": time.Since(start).Milliseconds(), "remote_addr": r.RemoteAddr, "user_agent": r.UserAgent()}
			if userID, ok := ctx.Value("userID").(string); ok {
				event["user_id"] = userID
			}
			body, _ := json.Marshal(event)
			logger.Print(string(body))
		})
	}
}
