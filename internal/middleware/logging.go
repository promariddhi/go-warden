package middleware

import (
	"log"
	"net/http"
	"time"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
)

func colorForStatus(status int) string {
	switch {
	case status >= 500:
		return red
	case status >= 400:
		return yellow
	case status >= 300:
		return blue
	default:
		return green
	}
}

type StatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *StatusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &StatusWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		start := time.Now()
		next.ServeHTTP(sw, r)
		duration := time.Since(start)

		color := colorForStatus(sw.status)

		log.Printf(
			"%s %s %s %s%d%s %v\n",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			color,
			sw.status,
			reset,
			duration,
		)
	})
}
