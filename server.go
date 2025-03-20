package main

import (
	"EDFS/p2p"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootStrapNodes    []string
}

type FileServer struct {
	FileServerOpts
	peerLock sync.Mutex
	peers    map[string]p2p.Peer
	store    *Store
	quitch   chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}
	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[p.RemoteAddr().String()] = p

	log.Printf("connected with remote %s", p.RemoteAddr())
	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("FileServer loop exited due to user action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println(err)
				return
			}

			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Println(err)
			}

		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handelMessageStoreFile(from, v)
	default:
		return fmt.Errorf("unknown message type: %T", msg.Payload)
	}
}

func (s *FileServer) handelMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) not found in peer map", from)
	}

	if _, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size)); err != nil {
		//ishhhhh
		return err
	}

	peer.(*p2p.TCPPeer).WG.Done()

	return nil

}

type Message struct {
	// From    string
	Payload any
}

// type DataMessage struct {
// 	Key  string
// 	Data []byte
// }

type MessageStoreFile struct {
	Key  string
	Size int64
}

func (s *FileServer) broadcast(msg *Message) error {
	var lastErr error
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	if len(s.peers) == 0 {
		return nil
	}

	for addr, peer := range s.peers {
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
			log.Printf("Failed to encode payload for peer %s: %v", addr, err)
			lastErr = err
			continue
		}

		if err := peer.Send(buf.Bytes()); err != nil {
			log.Printf("Failed to send to peer %s: %v", addr, err)
			lastErr = err
			//  remove the peer if it's unreachable
			delete(s.peers, addr)
			continue
		}
		log.Printf("Successfully broadcast to peer %s", addr)
	}

	return lastErr
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	// Read all data into buffer
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	// Write to local store first
	size, err := s.store.Write(key, tee)
	if err != nil {
		return err
	}

	fmt.Println("size of byte ", size)

	// Create control message
	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	// Lock peers map before accessing
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	// Skip if no peers
	if len(s.peers) == 0 {
		return nil
	}

	// Broadcast control message to all peers
	msgBuf := new(bytes.Buffer)
	if err := gob.NewEncoder(msgBuf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		if err := peer.Send(msgBuf.Bytes()); err != nil {
			return err
		}
	}

	// Now send the actual file data to each peer
	for addr, peer := range s.peers {
		n, err := io.Copy(peer, buf)
		if err != nil {
			log.Printf("Failed to send file data to peer %s: %v", addr, err)
			delete(s.peers, addr)
			continue
		}
		log.Printf("Successfully sent %d bytes to peer %s", n, addr)
	}

	return nil
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) bootstrapNetwork() error {

	for _, addr := range s.BootStrapNodes {
		// if len(addr) == 0 {
		// 	continue
		// }
		// fmt.Println("bootstrap")
		go func(addr string) {
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("Dail error: ", err)
			}
		}(addr)
	}
	return nil
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	s.bootstrapNetwork()
	s.loop()
	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
}
