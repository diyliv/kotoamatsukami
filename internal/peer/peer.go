package peer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"

	"github.com/diyliv/p2p/internal/client"
	"github.com/diyliv/p2p/internal/models"
	rsaenc "github.com/diyliv/p2p/pkg/rsa"
)

var (
	errInvalidCreds = errors.New("you didnt specify host or port")
)

type Peer struct {
	Connections           map[string][]string      `json:"connections"`
	Ip                    string                   `json:"peer_ip"`
	Port                  string                   `json:"peer_port"`
	ClientPrivateKey      *rsa.PrivateKey          `json:"peer_private_key"`
	ClientPublicKey       rsa.PublicKey            `json:"peer_public_key"`
	InterlocutorPublicKey rsa.PublicKey            `json:"interlocutor_key"`
	CheckKeys             map[string]rsa.PublicKey `json:"check_keys"`
	JoinedAt              time.Time                `json:"joined_at"`
	mu                    sync.Mutex
}

func NewPeer(addr string) (*Peer, error) {
	addrInfo := strings.Split(addr, ":")

	if len(addrInfo) != 2 {
		color.Red("You need to specify port.")
		return nil, errInvalidCreds
	}

	color.Magenta("[system] Generating keys.")
	keys, err := rsaenc.GenerateKeys()
	if err != nil {
		color.Red("[system error] Error while generating keys: %v\n", err)
		return nil, err
	}
	color.Green("[system] Keys were successfully generated.")

	return &Peer{
		Connections:      make(map[string][]string),
		Ip:               addrInfo[0],
		Port:             ":" + addrInfo[1],
		ClientPrivateKey: keys,
		ClientPublicKey:  keys.PublicKey,
		CheckKeys:        make(map[string]rsa.PublicKey),
		JoinedAt:         time.Now().Local(),
	}, nil
}

func (peer *Peer) Run(HandleServer func(*Peer), HandleClient func(*Peer)) {
	color.Magenta("[system] Starting listening on %s", os.Args[1])
	go HandleServer(peer)
	HandleClient(peer)
}

func (peer *Peer) LinkConnection(addrs []string) {
	peer.Connections[peer.Port] = addrs

	addrs[0] = strings.Trim(addrs[0], "\r\n")
	// color.Magenta("[system] Connecting to %s", addrs[0])

	conn, err := net.Dial("tcp", addrs[0])
	if err != nil {
		log.Println(err)
		color.Red("[system] Error while dialing with %s", addrs[0])
	}

	// color.Magenta("[system] Connection was successful. Sending our public key.")
	// color.Green("[system] Sending your public key was successful.")
	defer conn.Close()

	handshake := models.NewHandShake(peer.Port, peer.ClientPublicKey)
	m, err := json.Marshal(handshake)
	if err != nil {
		color.Red("[system] System error: %v\n", err)
		log.Println(err)
	}
	if _, err := conn.Write(m); err != nil {
		color.Red("[system] System error: %v\n", err)
		panic(err)
	}

	buf := make([]byte, 2048)

	l, err := conn.Read(buf)
	if err != nil {
		color.Red("[system] System error: %v\n", err)
		panic(err)
	}
	// color.Green("[system] Reading interlocutor public key was successful.")
	var respHandshake models.Handshake

	if err := json.Unmarshal(buf[:l], &respHandshake); err != nil {
		color.Red("[system] System error: %v\n", err)
		panic(err)
	}
	peer.mu.Lock()
	defer peer.mu.Unlock()
	peer.CheckKeys[respHandshake.Addr] = respHandshake.PublicKey
	peer.InterlocutorPublicKey = respHandshake.PublicKey

}

func (peer *Peer) SendMessageToAll(msg string) {
	var userMsg = &models.Message{
		From: peer.Ip + peer.Port,
		Body: []byte(msg),
	}

	peer.mu.Lock()
	val, ok := peer.Connections[peer.Port]
	if !ok {
		fmt.Println("you're not connected to any peer")
	}
	peer.mu.Unlock()

	for _, v := range val {
		userMsg.To = v
		if err := peer.Send(userMsg); err != nil {
			panic(err)
		}
	}

}

func (peer *Peer) Send(userMsg *models.Message) error {
	conn, err := net.Dial("tcp", userMsg.To)
	if err != nil {
		log.Println(err)
		return err
	}
	defer conn.Close()

	m, err := json.Marshal(userMsg)
	if err != nil {
		panic(err)
	}

	peer.mu.Lock()
	val, ok := peer.CheckKeys[userMsg.To]
	if !ok {
		panic(fmt.Sprintf("you didnt have public key from this user: %s", userMsg))
	}
	peer.mu.Unlock()

	encryptMsg, err := rsaenc.EncryptOAEP(sha256.New(), rand.Reader, &val, m)
	if err != nil {
		panic(err)
	}

	var answer models.Message
	answer.From = peer.Port
	answer.To = userMsg.To
	answer.Body = encryptMsg
	finM, err := json.Marshal(answer)
	if err != nil {
		panic(err)
	}

	if _, err := conn.Write(finM); err != nil {
		panic(err)
	}

	return nil
}

func (peer *Peer) AllPeers() {
	for _, v := range peer.Connections {
		fmt.Println("|", v)
	}
}

func HandleClient(peer *Peer) {
	for {
		msg := client.InputString()
		cmd := strings.Split(msg, " ")

		switch cmd[0] {
		case "/all":
			peer.AllPeers()
		case "/exit":
			os.Exit(0)
		case "/connect":
			peer.LinkConnection(cmd[1:])
		default:
			peer.SendMessageToAll(msg)
		}
	}
}

func HandleServer(peer *Peer) {
	lis, err := net.Listen("tcp", peer.Port)
	if err != nil {
		panic(err)
	}
	defer lis.Close()

	for {
		conn, err := lis.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(peer, conn)
	}
}

func handleConnection(peer *Peer, conn net.Conn) {
	buf := make([]byte, 2048)
	var msg string

	l, err := conn.Read(buf)
	if err != nil {
		conn.Close()
	}
	msg += string(buf[:l])
	var handshake models.Handshake

	if err := json.Unmarshal([]byte(msg), &handshake); err != nil {
		panic(err)
	}
	if handshake.Addr == "" {
		var userMsg models.Message

		if err := json.Unmarshal([]byte(msg), &userMsg); err != nil {
			panic(err)
		}
		peer.LinkConnection([]string{userMsg.From})
		decryptMsg, err := rsaenc.DecryptOAEP(sha256.New(), rand.Reader, peer.ClientPrivateKey, userMsg.Body)
		if err != nil {
			panic(err)
		}
		var resp models.Message

		if err := json.Unmarshal(decryptMsg, &resp); err != nil {
			panic(err)
		}

		fmt.Printf("[%s] %s\n", userMsg.From, string(resp.Body))
	}

	respHandshake := models.NewHandShake(peer.Port, peer.ClientPublicKey)

	m, err := json.Marshal(respHandshake)
	if err != nil {
		panic(err)
	}

	if _, err := conn.Write(m); err != nil {
		panic(err)
	}
	peer.mu.Lock()
	peer.CheckKeys[handshake.Addr] = handshake.PublicKey
	peer.mu.Unlock()
}
