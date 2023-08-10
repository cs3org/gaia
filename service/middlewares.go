package service

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/openzipkin/zipkin-go/idgenerator"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func RequestLoggerMiddleware(log *zerolog.Logger, next http.Handler) http.Handler {
	traceHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace := idgenerator.NewRandom128().TraceID()
			l := log.With().Str("trace_id", trace.String())
			next.ServeHTTP(w, r.WithContext(l.Logger().WithContext(r.Context())))
		})
	}

	requestHandler := hlog.AccessHandler(
		func(r *http.Request, status, size int, duration time.Duration) {
			log := hlog.FromRequest(r)
			var event *zerolog.Event
			switch {
			case status < 400:
				event = log.Info()
			case status < 500:
				event = log.Warn()
			default:
				event = log.Error()
			}

			event.Str("method", r.Method).
				Stringer("url", r.URL).
				Int("status", status).
				Int("size", size).
				Dur("duration", duration).
				Interface("request_header", r.Header).
				Send()
		},
	)

	return traceHandler(requestHandler(next))
}

func RecoverFromPanicMiddleware(log *zerolog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Msgf("panic: %v\n%s", r, debug.Stack())
			}
		}()
		next.ServeHTTP(w, r)
	})
}
