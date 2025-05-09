package handlers

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type WppWebhook struct {
	ctx         context.Context
	VerifyToken string
	FbClient    *firestore.Client
}

func NewWppWebHook(token string, client *firestore.Client) *WppWebhook {
	return &WppWebhook{context.Background(), token, client}
}

func (c *WppWebhook) handleVerify(w http.ResponseWriter, r *http.Request) {
	//
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == c.VerifyToken {
		fmt.Fprintf(w, "%s", challenge)
		log.Println("Webhook verified successfully")
	} else {
		http.Error(w, "Verification failed", http.StatusForbidden)
	}
}

// Webhook event handler
func (c *WppWebhook) handleWebhook(w http.ResponseWriter, r *http.Request) {
	//
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("Error decoding webhook payload: %v\n", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// save messages
	docRef, _, err := c.FbClient.Collection("wpp").Add(c.ctx, body)
	if err != nil {
		http.Error(w, fmt.Sprintf("handleWebhook add error %s", err.Error()), http.StatusBadRequest)
	}

	// return
	fmt.Fprintf(w, "doc: %s", docRef.ID)
}

func (c *WppWebhook) WppWebHook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		c.handleVerify(w, r)
	} else if r.Method == http.MethodPost {
		c.handleWebhook(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
