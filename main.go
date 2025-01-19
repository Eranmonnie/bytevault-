package main

import (
	"EDFS/p2p"
	"log"
	"time"
)

// func OnPeer(peer p2p.Peer) error {
// 	peer.Close()
// 	fmt.Println("peer connected")
// 	return nil
// }

func main() {
	tctTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO OnpeerFunc,
	}
	tcpTransport := p2p.NewTCPTransport(tctTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       "300_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         *tcpTransport,
	}
	s := NewFileServer(fileServerOpts)

	go func() {
		time.Sleep(3 * time.Second)
		s.Stop()
	}()

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}

}
