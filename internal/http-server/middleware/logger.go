package logger

import (
	"log"
	"net/http"
	"time"
)

func New(log *log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			next.ServeHTTP(w, r)

			log.Printf("%s - %s (%s) duration=%s", r.Method, r.URL.Path, r.RemoteAddr, time.Since(startTime).String())
		}

		return http.HandlerFunc(fn)
	}
}
