package handlers

import (
	"net/http"
)

func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	if r.Header.Get("HX-Request") != "" {
		w.Header().Set("HX-Redirect", url)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func HtmxRedirect(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("HX-Request") != "" {
		w.Header().Set("HX-Redirect", r.URL.String())
		return true
	}
	return false
}
