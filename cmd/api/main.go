package main

import (
	"os"

	"github.com/diyliv/p2p/internal/peer"
	"github.com/diyliv/p2p/pkg/logger"
)

func main() {
	logger := logger.InitLogger()

	p, err := peer.NewPeer(os.Args[1], logger)
	if err != nil {
		panic(err)
	}

	p.Run(peer.HandleServer, peer.HandleClient)
}
