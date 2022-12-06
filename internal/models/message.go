package models

type Message struct {
	From string `json:"from_peer"`
	To   string `json:"recipient_peer"`
	Body []byte `json:"body"`
}
