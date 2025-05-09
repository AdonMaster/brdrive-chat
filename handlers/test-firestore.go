package handlers

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"net/http"
)

func TestFirestore(fb *firestore.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//
		ctx := context.Background()
		docRef, _, err := fb.Collection("tests").Add(ctx, map[string]interface{}{
			"created_at": firestore.ServerTimestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//
		fmt.Fprintf(w, fmt.Sprintf("doc-id: %s", docRef.ID))
	}
}
