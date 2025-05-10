package wpp

import (
	"chat/helpers"
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Wpp struct {
	ctx         context.Context
	FbClient    *firestore.Client
	VerifyToken string
	AccessToken string
}

type ReadReceipt struct {
	MessagingProduct string `json:"messaging_product"`
	Status           string `json:"status"`
	MessageID        string `json:"message_id"`
}

type Message struct {
	ID            string    `json:"id"`
	Type          string    `json:"type,omitempty"`
	From          string    `json:"from,omitempty"`
	FromName      string    `json:"from_name,omitempty"`
	Timestamp     string    `json:"timestamp,omitempty"`
	TimestampTime time.Time `json:"timestampTime,omitempty"`
	Text          struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
}

type WebhookMessage struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaId string `json:"waId"`
				} `json:"contacts"`
				Messages []Message `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func NewWpp(client *firestore.Client) *Wpp {

	verifyToken := os.Getenv("WPP_VERIFY_TOKEN")
	accessToken := os.Getenv("WPP_ACCESS_TOKEN")

	//
	if verifyToken == "" {
		log.Fatalln("WPP_VERIFY_TOKEN not found")
	}
	if accessToken == "" {
		log.Fatalln("WPP_ACCESS_TOKEN not found")
	}

	//
	return &Wpp{
		context.Background(),
		client,
		verifyToken,
		accessToken,
	}
}

func (c *Wpp) handleVerify(w http.ResponseWriter, r *http.Request) {
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

func (c *Wpp) handleWebhook(w http.ResponseWriter, r *http.Request) {

	// save messages
	count, err := c.saveMessages(r.Body)
	if err != nil {
		log.Printf("erro %v", err)
	}

	// return
	fmt.Fprintf(w, "count: %d", count)
}

func (c *Wpp) saveMessages(body io.ReadCloser) (int, error) {

	//
	res := 0

	// read json binary
	jsonData, err := io.ReadAll(body)
	if err != nil {
		log.Println("wpp.go@saveMessages: erro ao ler json binary")
		return 0, err
	}
	defer body.Close()

	// try to parse into webhook message
	var msg WebhookMessage
	if err := json.Unmarshal(jsonData, &msg); err != nil {
		_, _, err2 := c.FbClient.Collection("wpp-unread").Add(c.ctx, map[string]string{
			"payload": string(jsonData),
			"err":     err.Error(),
		})
		if err2 != nil {
			log.Printf("wpp.go@saveMessages: Erro ao gravar wpp-unread: %v", err2)
		}
		return 0, err
	}

	// looping through it all
	for _, entry := range msg.Entry {
		for _, change := range entry.Changes {
			for _, message := range change.Value.Messages {

				// convert timestamp
				solveTimestamp(&message)
				solveFrom(msg, &message)

				// switch type
				switch message.Type {

				// text
				case "text":
					_, err := c.FbClient.Collection("wpp-message").Doc(message.ID).Set(c.ctx, message)
					if err != nil {
						return 0, err
					}
				}

				//
				res++
			}
		}
	}

	return res, nil
}

func solveTimestamp(message *Message) {
	unixTimestamp, err := strconv.ParseInt(message.Timestamp, 10, 64)
	if err != nil {
		message.TimestampTime = time.Now()
	} else {
		message.TimestampTime = time.Unix(unixTimestamp, 0)
	}
}

func solveFrom(parent WebhookMessage, msg *Message) {
	for _, p := range parent.Entry {
		for _, c := range p.Changes {
			for _, contact := range c.Value.Contacts {
				if contact.WaId == msg.From {
					msg.FromName = helpers.StrCoalesce(contact.Profile.Name, "-unknown-")
					return
				}
			}
		}
	}
	msg.FromName = "unknown"
}

func (c *Wpp) WppWebHook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		c.handleVerify(w, r)
	} else if r.Method == http.MethodPost {
		c.handleWebhook(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
