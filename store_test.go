package main

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestDeleteFunc(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	key := "test"
	data := []byte("testing")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}
func TestPathTransformFunc(t *testing.T) {
	key := "testingwow"
	expectedPathName := "c3228/a7b99/17ef0/0a2f3/9a92d/67593/7c5a5/088c4"
	expectedOriginalKey := "c3228a7b9917ef00a2f39a92d675937c5a5088c4"
	pathKey := CASPathTransformFunc(key)
	if pathKey.pathName != expectedPathName {
		t.Errorf("have %s, want %s", pathKey.pathName, expectedPathName)
	}
	if pathKey.fileName != expectedOriginalKey {
		t.Errorf("have %s, want %s", pathKey.fileName, expectedOriginalKey)
	}

}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	key := "test"
	data := []byte("testing")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if ok := s.Has(key); !ok {
		t.Errorf("Expected to hve key %s", key)
	}
	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, _ := ioutil.ReadAll(r)
	if string(b) != string(data) {
		t.Errorf("want %s, have%s", data, b)
	}

	s.Delete((key))
}
