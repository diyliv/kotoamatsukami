package main

import (
	"os"

	"github.com/diyliv/p2p/internal/peer"
)

func main() {
	p, err := peer.NewPeer(os.Args[1])
	if err != nil {
		panic(err)
	}

	p.Run(peer.HandleServer, peer.HandleClient)
}
