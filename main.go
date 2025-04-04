package main

import (
	"EDFS/p2p"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

//	func OnPeer(peer p2p.Peer) error {
//		peer.Close()
//		fmt.Println("peer connected")
//		return nil
//	}
func makeServer(ListenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    ListenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO OnpeerFunc,
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		EncKey:            newEncryptionKey(),
		StorageRoot:       strings.TrimPrefix(ListenAddr, ":") + "_network",
		Transport:         tcpTransport,
		BootStrapNodes:    nodes,
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer
	return s
}

func main() {

	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")
	go func() { log.Fatal(s1.Start()) }()
	go func() { log.Fatal(s2.Start()) }()

	time.Sleep(4 * time.Second)
	key := "key.jpg"
	data := bytes.NewReader([]byte("hello world"))
	if err := s2.Store(key, data); err != nil {
		log.Fatal(err)
	}

	if err := s2.store.Delete(s2.ID, key); err != nil {
		log.Fatal(err)
	}

	_, r, err := s2.Get(key)

	if err != nil {
		log.Fatal(err)
	}

	b, err := ioutil.ReadAll(r)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))
	// select {}
}
