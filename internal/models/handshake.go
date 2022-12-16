package models

import (
	"crypto/rsa"
	"time"
)

type Handshake struct {
	Addr      string        `json:"addr"`
	PublicKey rsa.PublicKey `json:"pub_key"`
	StartedAt time.Time     `json:"started_at"`
}

func NewHandShake(addr string, publicKey rsa.PublicKey) *Handshake {
	return &Handshake{
		Addr:      addr,
		PublicKey: publicKey,
		StartedAt: time.Now().Local(),
	}
}
