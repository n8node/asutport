package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func SlogMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimwWrapResponseWriter(w)
			next.ServeHTTP(ww, r)
			logger.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.status),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

type wrapResponseWriter struct {
	http.ResponseWriter
	status int
}

func chimwWrapResponseWriter(w http.ResponseWriter) *wrapResponseWriter {
	return &wrapResponseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (w *wrapResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
