package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
)

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
	s := newStore()
	id := generateID()
	defer tearDown(t, s)
	for i := 0; i < 1; i++ {
		key := fmt.Sprintf("test%d", i)
		data := []byte("testing")
		if _, err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := s.Has(id, key); !ok {
			t.Errorf("Expected to have key %s", key)
		}

		_, r, err := s.Read(id, key)
		if err != nil {
			t.Error(err)
		}

		// Add this block to ensure proper resource cleanup
		defer func() {
			if closer, ok := r.(io.Closer); ok {
				closer.Close()
			}
		}()

		b, _ := ioutil.ReadAll(r)
		if string(b) != string(data) {
			t.Errorf("want %s, have%s", data, b)
		}

		if closer, ok := r.(io.Closer); ok {
			closer.Close()
		}

		if err := s.Delete(id, key); err != nil {
			fmt.Println(err)
			t.Error(err)
		}

		if ok := s.Has(id, key); ok {
			t.Errorf("Expected to not have key %s", key)
		}
	}
}

func newStore() *Store {
	opts := StoreOpts{

		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)
}

func tearDown(t *testing.T, s *Store) {
	if err := s.clear(); err != nil {
		t.Error(err)
	}

}
