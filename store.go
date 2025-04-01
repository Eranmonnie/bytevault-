package main

import (
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
	Root string
	// ID of the owner of the storage which will be used to store all files at that lcation
	//so we can sync all files if needed

	PathTransformFunc PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		pathName: key,
		fileName: key,
	}
}

type Store struct {
	StoreOpts
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
func (s Store) Has(id string, key string) bool {
	PathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, PathKey.fullPath())
	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(id string, key string) error {
	pathKey := s.PathTransformFunc(key)
	defer func() {
		log.Printf("deleted [%s] from disc", pathKey.fileName)
	}()

	firstPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) openFile(id string, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.pathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	pathAndFileNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.fullPath())
	return os.Create(pathAndFileNameWithRoot)
}

func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
}

func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {

	f, err := s.openFile(id, key)
	if err != nil {
		return 0, err
	}

	n, err := copyDecrypt(encKey, r, f)
	if err != nil {
		log.Println("error in copy", err)
		f.Close()
		return 0, err
	}

	log.Printf("written (%d) bytes to disc : %s", n, f.Name())

	return int64(n), f.Close()

}

func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFile(id, key)
	if err != nil {
		return 0, err
	}

	n, err := io.Copy(f, r)
	if err != nil {
		log.Println("error in copy", err)
		f.Close()
		return 0, err
	}

	log.Printf("written (%d) bytes to disc : %s", n, f.Name())

	return n, f.Close()
}

func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
}

func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	pathKeyWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.fullPath())

	file, err := os.Open(pathKeyWithRoot)
	if err != nil {
		return 0, nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}
	return fi.Size(), file, nil
}
