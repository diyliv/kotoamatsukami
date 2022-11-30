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
	"time"

	"github.com/diyliv/p2p/internal/client"
	"github.com/diyliv/p2p/internal/models"
	rsaenc "github.com/diyliv/p2p/pkg/rsa"
)

var (
	errInvalidCreds = errors.New("you didnt specify host or port")
)

type Peer struct {
	Connections      map[string][]string `json:"connections"`
	Ip               string              `json:"peer_ip"`
	Port             string              `json:"peer_port"`
	ClientPrivateKey *rsa.PrivateKey     `json:"peer_private_key"`
	ClientPublicKey  rsa.PublicKey       `json:"peer_public_key"`
	writePrivKeyCh   chan bool
	JoinedAt         time.Time `json:"joined_at"`
}

func NewPeer(addr string) (*Peer, error) {
	addrInfo := strings.Split(addr, ":")

	if len(addrInfo) != 2 {
		return nil, errInvalidCreds
	}

	keys, err := rsaenc.GenerateKeys()
	if err != nil {
		return nil, err
	}

	return &Peer{
		Connections:      make(map[string][]string),
		Ip:               addrInfo[0],
		Port:             ":" + addrInfo[1],
		ClientPrivateKey: keys,
		ClientPublicKey:  keys.PublicKey,
		writePrivKeyCh:   make(chan bool, 1),
		JoinedAt:         time.Now().Local(),
	}, nil
}

func (peer *Peer) Run(HandleServer func(*Peer), HandleClient func(*Peer)) {
	log.Printf("starting listening on %s", os.Args[1])
	go HandleServer(peer)
	HandleClient(peer)
}

func (peer *Peer) LinkConnection(addrs []string) {
	peer.Connections[peer.Port] = addrs
}

func (peer *Peer) SendMessageToAll(msg string) {
	var userMsg = &models.Message{
		From: peer.Ip + peer.Port,
		Key:  peer.ClientPrivateKey,
		Body: msg,
	}

	val, ok := peer.Connections[peer.Port]
	if !ok {
		fmt.Println("you're not connected to any peer")
	}

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
		conn.Close()
		return err
	}
	defer conn.Close()

	var privKey struct {
		Key *rsa.PrivateKey `json:"key"`
		Msg []byte          `json:"msg"`
	}

	m, err := json.Marshal(userMsg)
	if err != nil {
		log.Println(err)
	}

	encData, err := rsaenc.EncryptOAEP(sha256.New(), rand.Reader, &peer.ClientPublicKey, m)
	if err != nil {
		log.Printf("Error while encrypting data: %v\n", err)
	}

	privKey.Key = peer.ClientPrivateKey
	privKey.Msg = encData

	finM, err := json.Marshal(privKey)
	if err != nil {
		panic(err)
	}

	if _, err := conn.Write(finM); err != nil {
		log.Println(err)
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
	lis, err := net.Listen("tcp", "192.168.1.9"+peer.Port)
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
	defer conn.Close()
	var msg string
	buff := make([]byte, 2048)

	var data models.Message

	for {
		length, err := conn.Read(buff)
		if err != nil {
			break
		}
		msg += string(buff[:length])
	}

	var resp struct {
		Key *rsa.PrivateKey `json:"key"`
		Msg []byte          `json:"msg"`
	}

	if err := json.Unmarshal([]byte(msg), &resp); err != nil {
		panic(err)
	}

	dec, err := rsaenc.DecryptOAEP(sha256.New(), rand.Reader, resp.Key, resp.Msg)
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(dec, &data); err != nil {
		panic(err)
	}

	peer.LinkConnection([]string{data.From})
	fmt.Printf("[%s]: %s\n", conn.RemoteAddr().String(), data.Body)
}
