package middleware

import "net/http"

// AbsoluteURL modifies requests so that the URL is absolute.
func AbsoluteURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Scheme == "" {
			r.URL.Scheme = "http"
		}

		r.URL.Host = r.Host

		next.ServeHTTP(w, r)
	})
}
