package peer

import (
	"errors"
	"strings"
)

var (
	invalidCreds = errors.New("you didnt specify host or port")
)

type Peer struct {
	Connections map[string][]string
	Ip          string
	Port        string
}

func (peer *Peer) NewPeer(addr string) (*Peer, error) {
	addrInfo := strings.Split(addr, ":")

	if len(addrInfo) != 2 {
		return nil, invalidCreds
	}
	return &Peer{
		Connections: make(map[string][]string),
		Ip:          addrInfo[0],
		Port:        ":" + addrInfo[1],
	}, nil
}
