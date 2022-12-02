package models

import (
	"crypto/rsa"
	"time"
)

type Handshake struct {
	PublicKey rsa.PublicKey `json:"pub_key"`
	StartedAt time.Time     `json:"started_at"`
}

func NewHandShake(publicKey rsa.PublicKey) *Handshake {
	return &Handshake{
		PublicKey: publicKey,
		StartedAt: time.Now().Local(),
	}
}
