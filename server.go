package main

import (
	"EDFS/p2p"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
)

type FileServerOpts struct {
	EncKey            []byte
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
		log.Println("FileServer loop exited due to error or user action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error:", err)
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

	case MessageGetFile:
		return s.handelMessageGetFile(from, v)
	default:
		return fmt.Errorf("unknown message type: %T", msg.Payload)
	}
}

func (s *FileServer) handelMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.Key) {
		return fmt.Errorf("file (%s) not found in local store", msg.Key)
	}

	fmt.Printf("got file (%s) requested by peer %s serving over the network \n", msg.Key, from)
	fileSize, r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		fmt.Println("closing file reader")

		defer rc.Close()
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) not found in peer map", from)
	}

	//first send incoming stream to peer and then send file size
	// as an int64
	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}
	fmt.Printf("file (%s) of size (%d) sent to peer %s over the network \n", msg.Key, n, from)
	return nil
}

func (s *FileServer) handelMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) not found in peer map", from)
	}

	log.Printf("Starting to read %d bytes from peer %s", msg.Size, from)

	if _, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size)); err != nil {
		return err
	}
	// log.Printf("stored data from peer %s", from)

	peer.CloseStream()

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

type MessageGetFile struct {
	Key string
}

type MessageRemoveFile struct {
	Key string
}

func (s *FileServer) Get(key string) (int64, io.Reader, error) {
	if s.store.Has(key) {
		return s.store.Read(key)
	}

	fmt.Printf("dont have file (%s) locally. Fetching from network... \n", key)
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return 0, nil, err
	}

	// time.Sleep(4 * time.Second)

	// read from peer

	for _, peer := range s.peers {
		// first read the filesize from binary read so
		// we can limit the ammount of byte we read from connection

		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)
		fmt.Println("file size ", fileSize)
		n, err := s.store.Write(key, io.LimitReader(peer, fileSize))

		if err != nil {
			return 0, nil, err
		}

		// fileBuf := new(bytes.Buffer)
		// n, err := io.CopyN(fileBuf, peer, 22)
		// if err != nil {
		// 	continue
		// }
		fmt.Printf("recieved (%d) bytes over the network from %s", n, peer.RemoteAddr())
		peer.CloseStream()
	}

	// select {}
	return s.store.Read(key)
}

func (s *FileServer) Remove(key string) error {
	if err := s.store.Delete(key); err != nil {
		return err
	}

	msg := Message{
		Payload: MessageRemoveFile{
			Key: key,
		},
	}

	return s.broadcast(&msg)
}

func (s *FileServer) stream(msg *Message) error {
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

func (s *FileServer) broadcast(msg *Message) error {
	msgBuf := new(bytes.Buffer)
	if err := gob.NewEncoder(msgBuf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(msgBuf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileServer) Store(key string, r io.Reader) error {
	// Read all data into buffer
	fileBuf := new(bytes.Buffer)
	tee := io.TeeReader(r, fileBuf)

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
			Size: size + 16,
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
	if err := s.broadcast(&msg); err != nil {
		return err
	}

	// Now send the actual file data to each peer
	for addr, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingStream})
		n, err := copyEncrypt(s.EncKey, fileBuf, peer)
		if err != nil {
			log.Printf("Failed to send file data to peer %s: %v", addr, err)
			delete(s.peers, addr)
			continue
		}
		// n, err := io.Copy(peer, fileBuf)
		// if err != nil {
		// 	log.Printf("Failed to send file data to peer %s: %v", addr, err)
		// 	delete(s.peers, addr)
		// 	continue
		// }
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
	gob.Register(MessageGetFile{})
}
