package main

import (
	"chat/handlers"
	"chat/mids"
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
	wppInstance := wpp.NewWpp(fbClient)
	http.HandleFunc("/wpp-webhook", wppInstance.WppWebHook)

	// wpp - handlers
	http.HandleFunc("/wpp", mids.Method(http.MethodPost, wppInstance.WppSend))

	//
	println(">> Listening to :4000...")
	_ = http.ListenAndServe(":4000", nil)
}
