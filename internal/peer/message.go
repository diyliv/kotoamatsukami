package peer

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"net"

	"github.com/diyliv/p2p/internal/models"
	rsaenc "github.com/diyliv/p2p/pkg/rsa"
	"github.com/fatih/color"
)

func (peer *Peer) SendMessageToAll(msg string) {
	var userMsg = &models.Message{
		From: peer.Ip + peer.Port,
		Body: []byte(msg),
	}

	if msg == "" {
		return
	}

	peer.mu.Lock()
	val := peer.Connections[peer.Port]
	if len(val) == 0 {
		color.Yellow("You're not connected to any peer.")
	}

	connections := peer.removeDuplicates(val)
	peer.Connections[peer.Port] = connections
	peer.mu.Unlock()

	for _, v := range connections {
		userMsg.To = v
		if err := peer.Send(userMsg); err != nil {
			color.Red("[system] error while sending message: %v\n", err)
			peer.logger.Error("Error while sending message: " + err.Error())
		}
	}

}

func (peer *Peer) Send(userMsg *models.Message) error {
	conn, err := net.Dial("tcp", userMsg.To)
	if err != nil {
		color.Red("[system] %s disconnected\n", userMsg.To)
		peer.mu.Lock()
		defer peer.mu.Unlock()

		connections := peer.Connections[peer.Port]
		for idx, val := range connections {
			if val == userMsg.To {
				updatedConnections := peer.removeElement(peer.Connections[peer.Port], idx)
				peer.Connections[peer.Port] = updatedConnections
			}
		}

		return err
	}
	defer conn.Close()

	m, err := json.Marshal(userMsg)
	if err != nil {
		peer.logger.Error("Error while marshalling message: " + err.Error())
		return err
	}

	peer.mu.Lock()
	val, ok := peer.CheckKeys[userMsg.To]
	if !ok {
		peer.logger.Error("Kinda strange. You dont have public key from this user: " + userMsg.To)
	}
	peer.mu.Unlock()

	encryptMsg, err := rsaenc.EncryptOAEP(sha256.New(), rand.Reader, &val, m)
	if err != nil {
		peer.logger.Error("Error while encrypting your message: " + err.Error())
		return err
	}

	var answer models.Message
	answer.From = peer.Port
	answer.To = userMsg.To
	answer.Body = encryptMsg

	finM, err := json.Marshal(answer)
	if err != nil {
		peer.logger.Error("Error while marshalling message: " + err.Error())
		return err
	}

	if _, err := conn.Write(finM); err != nil {
		peer.logger.Error("Error while writing mesasge: " + err.Error())
		return err
	}

	return nil
}

func (peer *Peer) DirectMessage(addr, msg string) {
	var userMsg = models.Message{
		From: peer.Port,
		To:   addr,
		Body: []byte(msg),
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		color.Red("[system] Host unavailable.")
		peer.logger.Error("Error while connecting: " + err.Error())
	}
	defer conn.Close()

	peer.mu.Lock()
	defer peer.mu.Unlock()
	_, ok := peer.CheckKeys[addr]
	if !ok {
		color.Magenta("[system] Didn't find keys for this host. Handshaking...")
		if err := peer.LinkConnection([]string{addr}); err != nil {
			color.Red("[system] Error while handshaking: " + err.Error())
			peer.logger.Error("Error while handshaking: " + err.Error())
		}
		color.Magenta("[system] Handshaking was successful.")
	}

	err = peer.Send(&userMsg)
	if err != nil {
		color.Red("[system] error while sending message: %v\n", err)
		peer.logger.Error("Error while sending message: " + err.Error())
	} else {
		conn.Close()
	}
}
