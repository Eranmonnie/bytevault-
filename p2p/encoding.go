package p2p

import (
	"encoding/gob"

	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecder struct{}

func (dec GOBDecder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	buff := make([]byte, 1028)
	n, err := r.Read(buff)
	if err != nil {
		return err
	}
	msg.Payload = buff[:n]

	return nil

}
