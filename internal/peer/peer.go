package peer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	mathrand "math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"

	"github.com/diyliv/p2p/internal/client"
	"github.com/diyliv/p2p/internal/models"
	httpserver "github.com/diyliv/p2p/internal/server"
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
	logger                *zap.Logger
}

func NewPeer(addr string, logger *zap.Logger) (*Peer, error) {
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
	color.Magenta("[system] Keys were successfully generated.")

	return &Peer{
		Connections:      make(map[string][]string),
		Ip:               addrInfo[0],
		Port:             ":" + addrInfo[1],
		ClientPrivateKey: keys,
		ClientPublicKey:  keys.PublicKey,
		CheckKeys:        make(map[string]rsa.PublicKey),
		JoinedAt:         time.Now().Local(),
		logger:           logger,
	}, nil
}

func (peer *Peer) Run(HandleServer func(*Peer), HandleClient func(*Peer)) {
	color.Magenta("[system] Starting listening on %s", os.Args[1])
	color.Magenta("[system] Available commands\n/all - lists all your connections\n/connect [:port] - make connection with some peer\n/exit - quit the program\n/upload - start http server and upload file\n")
	go HandleServer(peer)
	HandleClient(peer)
}

func (peer *Peer) removeDuplicates(slice []string) []string {
	allKeys := make(map[string]bool)

	resultSlice := make([]string, 0)

	for _, value := range slice {
		if _, v := allKeys[value]; !v {
			allKeys[value] = true
			resultSlice = append(resultSlice, value)
		}
	}

	return resultSlice
}

func (peer *Peer) removeElement(slice []string, idx int) []string {
	slice[idx] = slice[len(slice)-1]
	return slice[:len(slice)-1]
}

func (peer *Peer) LowerUpper(str string) string {
	var res string

	mathrand.Seed(time.Now().UnixNano())

	for i := 0; i < len(str); i++ {
		a := mathrand.Intn(100)

		if a < 50 {
			newStr := strings.ToUpper(string(str[i]))
			res += newStr
		} else {
			newStr := strings.ToLower(string(str[i]))
			res += newStr
		}
	}
	return res
}

func (peer *Peer) LinkConnection(addrs []string) {
	var conn net.Conn
	var err error

	if addrs[0] == peer.Port {
		underline := color.New(color.FgRed).Add(color.Underline)
		color.Magenta("[system] You're trying to connect to %s\n", underline.Sprintf("Yourself. %s", peer.LowerUpper("crazy isn`t it?")))
		return
	}

	peer.mu.Lock()
	for i := 0; i < len(addrs); i++ {
		addrs[i] = strings.Trim(addrs[i], "\r\n")
		conn, err = net.Dial("tcp", addrs[i])
		if err != nil {
			peer.logger.Sugar().Errorf("Error while dialing with: %s. Error: %v\n", addrs[i], err)
			color.Red("[system] Error while dialing with %s", addrs[i])
			break
		}
		peer.Connections[peer.Port] = append(peer.Connections[peer.Port], addrs...)
	}
	peer.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			color.Magenta("[system] Looks like the peer you're trying to connect is offline. Try another one.")
		}
	}()

	defer conn.Close()

	handshake := models.NewHandShake(peer.Port, peer.ClientPublicKey)

	m, err := json.Marshal(handshake)
	if err != nil {
		color.Red("[system] System error: %v\n", err)
		peer.logger.Error("Error while marshalling key: " + err.Error())
	}

	if _, err := conn.Write(m); err != nil {
		color.Red("[system] System error: %v\n", err)
		peer.logger.Error("Error while writing key: " + err.Error())
	}

	buf := make([]byte, 2048)

	l, err := conn.Read(buf)
	if err != nil {
		color.Red("[system] System error: %v\n", err)
		peer.logger.Error("Error while reading response from peer: " + err.Error())
	}

	var respHandshake models.Handshake

	if err := json.Unmarshal(buf[:l], &respHandshake); err != nil {
		color.Red("[system] System error: %v\n", err)
		peer.logger.Error("Error while unmarshalling response: " + err.Error())
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

func (peer *Peer) AllPeers() {
	peer.mu.Lock()
	connections := peer.Connections[peer.Port]
	if len(connections) == 0 {
		color.Blue("No connections.")
	} else {
		color.Blue(fmt.Sprintf("|%s\n", peer.removeDuplicates(peer.Connections[peer.Port])))
	}
	peer.mu.Unlock()
}

func HandleClient(peer *Peer) {
	for {
		msg := client.InputString()
		cmd := strings.Split(msg, " ")

		switch cmd[0] {
		case "/all":
			peer.AllPeers()
		case "/exit":
			color.Magenta("[system] see ya :)")
			os.Exit(0)
		case "/connect":
			peer.LinkConnection(cmd[1:])
		case "/upload":
			go httpserver.NewServer(peer.logger).StartHTTP()
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
			peer.logger.Error("Error while accepting connection: " + err.Error())
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
		peer.logger.Error("Error while unmarshalling key: " + err.Error())
	}

	if handshake.Addr == "" {
		var userMsg models.Message

		if err := json.Unmarshal([]byte(msg), &userMsg); err != nil {
			peer.logger.Error("Error while unmarshalling message: " + err.Error())
		}

		peer.LinkConnection([]string{userMsg.From})

		decryptMsg, err := rsaenc.DecryptOAEP(sha256.New(), rand.Reader, peer.ClientPrivateKey, userMsg.Body)
		if err != nil {
			peer.logger.Error("Error while decrypting message: " + err.Error())
		}
		var resp models.Message

		if err := json.Unmarshal(decryptMsg, &resp); err != nil {
			peer.logger.Error("Error while unmarshalling message:  " + err.Error())
		}

		color.Cyan("[%s] %s\n", userMsg.From, string(resp.Body))
	}

	respHandshake := models.NewHandShake(peer.Port, peer.ClientPublicKey)

	m, err := json.Marshal(respHandshake)
	if err != nil {
		peer.logger.Error("Error while marshalling handshake:  " + err.Error())
	}

	if _, err := conn.Write(m); err != nil {
		peer.logger.Error("Error while writing message: " + err.Error())
	}
	peer.mu.Lock()
	peer.CheckKeys[handshake.Addr] = handshake.PublicKey
	peer.mu.Unlock()
}
