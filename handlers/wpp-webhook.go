package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type WppWebhook struct {
	VerifyToken string
}

func NewWppWebHook(token string) *WppWebhook {
	return &WppWebhook{token}
}

func (c *WppWebhook) handleVerify(w http.ResponseWriter, r *http.Request) {
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
func (*WppWebhook) handleWebhook(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("Error decoding webhook payload: %v\n", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Received webhook: %+v\n", body)

	// You can extract messages here
	// Example: body["entry"][0]["changes"][0]["value"]["messages"]

	w.WriteHeader(http.StatusOK)
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
