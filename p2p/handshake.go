package p2p

import "errors"

//returnde if the handshake between local and remote node cant be established
var ErrInvalidHandshake = errors.New("invalid handshake")

// handshake func  is  ...?
type HandshakeFunc func(Peer) error

func NOPHandshakeFunc(Peer) error { return nil }
