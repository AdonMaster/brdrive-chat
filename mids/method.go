package mids

import "net/http"

func Method(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method {
			next(w, r)
			return
		}
		http.Error(w, "Ops! Method not allowed", http.StatusMethodNotAllowed)
	}
}
