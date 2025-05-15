package responses

import (
	json2 "encoding/json"
	"net/http"
)

type Statusable interface {
	Status() int
}

//

type Ok struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func MakeOk(message string) *Ok {
	return &Ok{
		Status:  200,
		Message: message,
	}
}

type Err struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Payload interface{} `json:"payload"`
}

func MakeErr(status int, message string, payload ...interface{}) *Err {
	var pl interface{}
	if len(payload) > 0 {
		pl = payload[0]
	}
	return &Err{
		Status:  status,
		Message: message,
		Payload: pl,
	}
}
func MakeErrDef(message string, payload ...interface{}) *Err {
	return MakeErr(400, message, payload...)
}

type Payload struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Payload any    `json:"payload"`
}

func MakePayload(message string, payload any) Payload {
	return Payload{
		Status:  200,
		Message: message,
		Payload: payload,
	}
}

// ----- sugar
func respond(w http.ResponseWriter, status int, jsonable interface{}) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	data, err := json2.Marshal(jsonable)
	if err != nil {
		panic(err)
	}
	w.Write(data)
}

func (o *Ok) Write(w http.ResponseWriter) {
	respond(w, o.Status, o)
}
func (e *Err) Write(w http.ResponseWriter) {
	respond(w, e.Status, e)
}
func (p *Payload) Write(w http.ResponseWriter) {
	respond(w, p.Status, p)
}
