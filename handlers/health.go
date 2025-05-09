package handlers

import (
	"fmt"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, "Kiss me!")
}
