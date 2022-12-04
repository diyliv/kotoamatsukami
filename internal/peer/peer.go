package peer

import (
	"crypto/rsa"
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
	Connections           map[string][]string      `json:"connections"`
	Ip                    string                   `json:"peer_ip"`
	Port                  string                   `json:"peer_port"`
	ClientPrivateKey      *rsa.PrivateKey          `json:"peer_private_key"`
	ClientPublicKey       rsa.PublicKey            `json:"peer_public_key"`
	InterlocutorPublicKey rsa.PublicKey            `json:"interlocutor_key"`
	CheckKeys             map[string]rsa.PublicKey `json:"check_keys"`
	JoinedAt              time.Time                `json:"joined_at"`
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
		CheckKeys:        make(map[string]rsa.PublicKey),
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

	conn, err := net.Dial("tcp", addrs[0])
	if err != nil {
		log.Println(err)
	}

	defer conn.Close()

	handshake := models.NewHandShake(peer.Port, peer.ClientPublicKey)
	m, err := json.Marshal(handshake)
	if err != nil {
		log.Println(err)
	}
	if _, err := conn.Write(m); err != nil {
		panic(err)
	}
}

func (peer *Peer) SendMessageToAll(msg string) {
	var userMsg = &models.Message{
		From: peer.Ip + peer.Port,
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
		log.Println(err)
		return err
	}
	defer conn.Close()

	m, err := json.Marshal(userMsg)
	if err != nil {
		panic(err)
	}

	if _, err := conn.Write(m); err != nil {
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

	conn, err := lis.Accept()
	if err != nil {
		panic(err)
	}
	go handleConnection(peer, conn)
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
	fmt.Println(msg)

	if err := json.Unmarshal([]byte(msg), &data); err != nil {
		panic(err)
	}

	peer.LinkConnection([]string{data.From})
	fmt.Printf("[%s]: %s\n", conn.RemoteAddr().String(), data.Body)
}
