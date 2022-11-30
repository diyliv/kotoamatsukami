package models

import "crypto/rsa"

type Message struct {
	Key  *rsa.PrivateKey `json:"key"`
	From string          `json:"from_peer"`
	To   string          `json:"recipient_peer"`
	Body string          `json:"body"`
}
