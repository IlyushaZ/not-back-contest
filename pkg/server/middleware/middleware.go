package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("panic caught",
					slog.String("method", r.Method),
					slog.String("request_uri", r.URL.RequestURI()),
					slog.Any("panic", p),
					slog.String("stacktrace", string(debug.Stack())),
				)

				if slog.Default().Enabled(nil, slog.LevelDebug) {
					debug.PrintStack()
				}

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter

	status  int
	delay   time.Duration
	written int
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written = n
	return n, err
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if slog.Default().Enabled(nil, slog.LevelDebug) {
			schema := "http"
			if r.TLS != nil {
				schema = "https"
			}

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			now := time.Now()

			next.ServeHTTP(rw, r)

			slog.Debug("request served",
				slog.Duration("delay", time.Since(now)),
				slog.String("method", r.Method),
				slog.String("schema", schema),
				slog.String("uri", r.URL.RequestURI()),
				slog.Any("headers", r.Header),
				slog.Int("status", rw.status),
				slog.Int("response_length", rw.written),
			)

			return
		}

		next.ServeHTTP(w, r)
	})
}
