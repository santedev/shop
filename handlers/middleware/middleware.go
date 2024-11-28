package middleware

import (
	"log/slog"
	"net/http"
)

type HTTPHandler func(w http.ResponseWriter, r *http.Request) error

func LogErr(h HTTPHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {			
			slog.Error("HTTP handler error", "err", err, "path", r.URL.Path)
		}
	}
}

func LogErrAndRedirect(h HTTPHandler, redirectUrl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			ok := len(redirectUrl) > 0 
			slog.Error("HTTP handler error", "err", err, "path", r.URL.Path)
			if ok {
				w.Header().Set("Location", redirectUrl)
				w.WriteHeader(http.StatusSeeOther)
			}
		}
	}
}
