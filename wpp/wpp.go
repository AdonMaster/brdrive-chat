package wpp

import (
	"bytes"
	"chat/helpers"
	"chat/responses"
	"chat/validator"
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
	ctx          context.Context
	FbClient     *firestore.Client
	VerifyToken  string
	AccessToken  string
	cacheContact map[string]string
	cacheAccount map[string]string
}

type ReadReceipt struct {
	MessagingProduct string `json:"messaging_product"`
	Status           string `json:"status"`
	MessageID        string `json:"message_id"`
}

type SendMessage struct {
	MessagingProduct string          `json:"messaging_product"`
	To               string          `json:"to"`
	Type             string          `json:"type"`
	Text             SendMessageText `json:"text"`
}
type SendMessageText struct {
	Body string `json:"body"`
}

type Message struct {
	ID            string    `json:"id"`
	Account       string    `json:"account,omitempty"`
	AccountNo     string    `json:"account_no,omitempty"`
	Type          string    `json:"type,omitempty"`
	From          string    `json:"from,omitempty"`
	FromName      string    `json:"from_name,omitempty"`
	Timestamp     string    `json:"timestamp,omitempty"`
	TimestampTime time.Time `json:"timestampTime,omitempty"`
	Text          struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
}

type Contact struct {
	ID                   string    `json:"id"`
	Account              string    `json:"account"`
	Display              string    `json:"display,omitempty"`
	LastMessage          string    `json:"last_message,omitempty"`
	LastMessageTimestamp time.Time `json:"last_message_timestamp"`
}

type Account struct {
	ID    string `json:"id"`
	Phone string `json:"phone"`
}

type WebhookMessage struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Metadata struct {
					DisplayPhoneNumber string `json:"display_phone_number,omitempty"`
					PhoneNumberId      string `json:"phone_number_id,omitempty"`
				} `json:"metadata"`
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaId string `json:"wa_id"`
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
		make(map[string]string),
		make(map[string]string),
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
	count, err := c.saveBody(r.Body)
	if err != nil {
		log.Printf("erro %v", err)
	}

	// return
	fmt.Fprintf(w, "count: %d", count)
}

func (c *Wpp) saveBody(body io.ReadCloser) (int, error) {

	//
	res := 0

	// read json binary
	jsonData, err := io.ReadAll(body)
	if err != nil {
		log.Println("wpp.go@saveBody: erro ao ler json binary")
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
			log.Printf("wpp.go@saveBody: Erro ao gravar wpp-unread: %v", err2)
		}
		return 0, err
	}

	// looping through it all
	for _, entry := range msg.Entry {
		for _, change := range entry.Changes {
			for _, message := range change.Value.Messages {

				// assign
				message.Account = change.Value.Metadata.PhoneNumberId
				message.AccountNo = change.Value.Metadata.DisplayPhoneNumber

				// convert timestamp
				c.solveTimestamp(&message)
				c.solveFrom(msg, &message)

				// switch type
				switch message.Type {

				// text
				case "text":
					err := c.saveMessageText(message)
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

func (c *Wpp) saveMessageText(message Message) error {

	// message
	println("==> writing wpp-messages")
	_, err := c.FbClient.Collection("wpp-messages").Doc(message.ID).Set(c.ctx, message)
	if err != nil {
		return err
	}

	// account
	accountCached := c.cacheAccount[message.Account]
	if accountCached != message.AccountNo {
		println("==> writing wpp-accounts")
		_, err := c.FbClient.Collection("wpp-accounts").
			Doc(message.Account).
			Set(c.ctx, Account{
				ID:    message.Account,
				Phone: message.AccountNo,
			})
		if err != nil {
			return err
		}
		c.cacheAccount[message.Account] = message.AccountNo
	}

	// contact
	contactSaved := false
	if message.FromName != "" {
		contactCached := c.cacheContact[message.From]
		if contactCached != message.FromName {
			println("==> setting wpp-contacts")
			_, err = c.FbClient.Collection("wpp-contacts").
				Doc(message.From).
				Set(c.ctx, Contact{
					ID:                   message.From,
					Account:              message.Account,
					Display:              message.FromName,
					LastMessage:          message.Text.Body,
					LastMessageTimestamp: message.TimestampTime,
				})
			if err != nil {
				return err
			}
			c.cacheContact[message.From] = message.FromName
			contactSaved = true
		}
	}
	if !contactSaved {
		println("==> updating wpp-contacts")
		_, err = c.FbClient.Collection("wpp-contacts").
			Doc(message.From).
			Set(c.ctx, map[string]interface{}{
				"id":                     message.From,
				"last_message":           message.Text.Body,
				"last_message_timestamp": message.Timestamp,
			}, firestore.MergeAll)
		if err != nil {
			return err
		}
	}

	//
	return nil
}

func (c *Wpp) solveTimestamp(message *Message) {
	unixTimestamp, err := strconv.ParseInt(message.Timestamp, 10, 64)
	if err != nil {
		message.TimestampTime = time.Now()
	} else {
		message.TimestampTime = time.Unix(unixTimestamp, 0)
	}
}

func (c *Wpp) solveFrom(parent WebhookMessage, msg *Message) {
	for _, p := range parent.Entry {
		for _, change := range p.Changes {
			for _, contact := range change.Value.Contacts {
				if contact.WaId == msg.From {
					msg.FromName = helpers.StrCoalesce(contact.Profile.Name, c.cacheContact[msg.From])
					return
				}
			}
		}
	}
	msg.FromName = ""
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

func (c *Wpp) WppSend(w http.ResponseWriter, r *http.Request) {
	//
	var data struct {
		Account string `json:"account" v:"required"`
		Phone   string `json:"phone" v:"required"`
		Body    string `json:"body" v:"required"`
	}
	if !validator.Validate(w, r, &data) {
		return
	}

	//
	msg := SendMessage{
		MessagingProduct: "whatsapp",
		To:               data.Phone,
		Type:             "text",
		Text:             SendMessageText{data.Body},
	}
	_, err := json.Marshal(msg)
	if err != nil {
		responses.MakeErrDef(err.Error()).Write(w)
		return
	}

	//
	url := fmt.Sprintf("https://graph.facebook.com/v22.0/611404562060697/messages")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{
		"messaging_product": "whatsapp",
		"to": "5562984389210",
		"text": {
			"body": "simple"
		}
	}`))
	if err != nil {
		responses.MakeErrDef(err.Error()).Write(w)
		return
	}

	//
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	//
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		responses.MakeErrDef(err.Error()).Write(w)
		return
	}
	defer resp.Body.Close()
	//
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		responses.MakeErrDef(err.Error()).Write(w)
		return
	}

	//
	if resp.StatusCode >= 400 {
		responses.
			MakeErr(resp.StatusCode, "erro ao enviar", string(responseBytes)).
			Write(w)
		return
	}

	//
	responses.MakePayload("ok", string(responseBytes)).Write(w)
}
