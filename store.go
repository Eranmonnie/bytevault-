package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "ggnetwork"

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = hashStr[from:to]
	}
	return PathKey{pathName: strings.Join(paths, "/"), fileName: hashStr}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	pathName string
	fileName string
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.pathName, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (p PathKey) fullPath() string {
	return fmt.Sprintf("%s/%s", p.pathName, p.fileName)
}

type StoreOpts struct {
	// Root is the folder name of the root , containing all the folders/files of the system
	Root              string
	PathTransformFunc PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		pathName: key,
		fileName: key,
	}
}

type Store struct {
	StoreOpts StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}
	return &Store{
		StoreOpts: opts,
	}
}
func (s Store) Has(key string) bool {
	PathKey := s.StoreOpts.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.StoreOpts.Root, PathKey.fullPath())
	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) clear() error {
	return os.RemoveAll(s.StoreOpts.Root)
}

func (s *Store) Delete(key string) error {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	defer func() {
		log.Printf("deleted [%s] from disc", pathKey.fileName)
	}()

	firstPathNameWithRoot := fmt.Sprintf("%s/%s", s.StoreOpts.Root, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buff := new(bytes.Buffer)
	_, err = io.Copy(buff, f)

	return buff, err
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	pathKeyWithRoot := fmt.Sprintf("%s/%s", s.StoreOpts.Root, pathKey.fullPath())
	return os.Open(pathKeyWithRoot)
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.StoreOpts.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s", s.StoreOpts.Root, pathKey.pathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return err
	}

	pathAndFileNameWithRoot := fmt.Sprintf("%s/%s", s.StoreOpts.Root, pathKey.fullPath())
	f, err := os.Create(pathAndFileNameWithRoot)

	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)
	if err != nil {
		f.Close()
		return err
	}

	log.Printf("written (%d) bytes to disc : %s", n, pathAndFileNameWithRoot)

	return f.Close()
}
