package main

import (
	"chat/handlers"
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
	clientFb, err := firestore.NewClient(ctx, "brdrive-6c0c3")
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}
	defer clientFb.Close()

	// miscs
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/health", handlers.HealthHandler)

	// tests
	http.HandleFunc("/test-firestore", handlers.TestFirestore(clientFb))

	// wpp - webhook
	wppWebhookHandler := handlers.NewWppWebHook("__c9a68ea80")
	http.HandleFunc("/wpp-webhook", wppWebhookHandler.WppWebHook)

	//
	println(">> Listening to :4000...")
	_ = http.ListenAndServe(":4000", nil)
}
