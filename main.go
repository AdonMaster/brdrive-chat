package main

import (
	"chat/handlers"
	"chat/wpp"
	"cloud.google.com/go/firestore"
	"context"
	_ "embed"
	"log"
	"net/http"
)

func main() {

	//
	ctx := context.Background()

	// fb client
	fbClient, err := firestore.NewClient(ctx, "brdrive-6c0c3")
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}
	defer fbClient.Close()

	// miscs
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/health", handlers.HealthHandler)

	// tests
	http.HandleFunc("/test-firestore", handlers.TestFirestore(fbClient))

	// wpp - webhook
	http.HandleFunc("/wpp-webhook", wpp.NewWpp(fbClient).WppWebHook)

	//
	println(">> Listening to :4000...")
	_ = http.ListenAndServe(":4000", nil)
}
