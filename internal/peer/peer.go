package peer

import (
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
)

var (
	errInvalidCreds = errors.New("you didnt specify host or port")
)

type Peer struct {
	Connections map[string][]string
	connections []string
	Ip          string
	Port        string
	JoinedAt    time.Time
}

func NewPeer(addr string) (*Peer, error) {
	addrInfo := strings.Split(addr, ":")

	if len(addrInfo) != 2 {
		return nil, errInvalidCreds
	}
	return &Peer{
		Connections: make(map[string][]string),
		Ip:          addrInfo[0],
		Port:        ":" + addrInfo[1],
		JoinedAt:    time.Now().Local(),
	}, nil
}

func (peer *Peer) Run(HandleServer func(*Peer), HandleClient func(*Peer)) {
	log.Printf("starting listening on %s", os.Args[1])
	go HandleServer(peer)
	HandleClient(peer)
}

func (peer *Peer) LinkConnection(addrs []string) {
	fmt.Println("generating keys")
	peer.Connections[peer.Port] = append(peer.Connections[peer.Port], addrs...)
	// peer.Connections[peer.Port] = append(peer.Connections[peer.Port], addrs)
	os.Stdin.Write([]byte("connecting\n"))
}

func (peer *Peer) SendMessageToAll(msg string) {
	var userMsg = &models.Message{
		From: peer.Ip + peer.Port,
		Body: msg,
	}

	val, ok := peer.Connections[peer.Port]
	if !ok {
		fmt.Println("you're not connectd to any peer")
	}

	for idx, value := range val {
		userMsg.To = value
		if err := peer.Send(userMsg); err != nil {
			val[idx] = val[len(val)-1]
			log.Printf("unable to connect: %v\n", err)
			break
		}
		log.Println("reconnecting...")
	}
	val = val[:len(val)-1]
	fmt.Println(val)
	peer.Send(userMsg)
}

func (peer *Peer) Send(userMsg *models.Message) error {
	conn, err := net.Dial("tcp", userMsg.To)
	if err != nil {
		conn.Close()
		return err
	}
	defer conn.Close()

	m, err := json.Marshal(*userMsg)
	if err != nil {
		log.Println(err)
		return err
	}

	if _, err := conn.Write([]byte(m)); err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (peer *Peer) AllPeers() {
	for v := range peer.Connections {
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
	defer conn.Close()
	log.Printf("got connection from %s\n", conn.RemoteAddr().String())
	var msg string
	buff := make([]byte, 1024)
	var data *models.Message

	for {
		len, err := conn.Read(buff)
		if err != nil {
			break
		}

		msg += string(buff[:len])
		if err := json.Unmarshal([]byte(msg), &data); err != nil {
			panic(err)
		}
		peer.LinkConnection([]string{data.From})
		fmt.Println(data.Body)
	}
}
