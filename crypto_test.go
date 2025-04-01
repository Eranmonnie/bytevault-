package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestNewEncryptionKey(t *testing.T) {
	key := newEncryptionKey()
	if len(key) != 32 {
		t.Error("Expected key length of 32")
	}
}
func TestCopyEcrypt(t *testing.T) {

	src := bytes.NewReader([]byte("test"))
	dst := new(bytes.Buffer)
	key := newEncryptionKey()
	_, err := copyEncrypt(key, src, dst)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("Encrypted data: ", dst.Bytes())
}

func TestCopyDecrypt(t *testing.T) {

	payload := "test"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key := newEncryptionKey()
	_, err := copyEncrypt(key, src, dst)
	if err != nil {
		t.Error(err)
	}

	out := new(bytes.Buffer)
	_, err = copyDecrypt(key, dst, out)
	if err != nil {
		t.Error(err)
	}

	if out.String() != payload {
		t.Error("Expected decrypted data to be equal to the original data")
	}
}
