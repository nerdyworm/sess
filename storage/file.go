package storage

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"strings"
)

type FileStore struct {
	root string
}

func NewFileStore(root string) FileStore {
	return FileStore{root}
}

func (s FileStore) Put(key string, reader io.Reader) error {
	path := s.makePath(key)
	file := s.makeFile(key)

	err := os.MkdirAll(path, 0777)
	if err != nil {
		return err
	}

	writer, err := os.Create(file)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	return err
}

func (s FileStore) Exists(key string) (bool, error) {
	path := s.makeFile(key)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
	}

	return true, nil
}

func (s FileStore) Get(key string) (io.ReadCloser, error) {
	file := s.makeFile(key)
	return os.Open(file)
}

func (s FileStore) GetPath(key string) string {
	file := s.makeFile(key)
	return file
}

func (s FileStore) Delete(key string) error {
	file := s.makeFile(key)
	return os.Remove(file)
}

func (s FileStore) makePath(key string) string {
	return makeRoot(s.root, key)
}

func (s FileStore) makeFile(key string) string {
	return makeFile(s.root, key)
}

func makeRoot(root, key string) string {
	parts := strings.Split(key, "/")
	return root + strings.Join(parts[:len(parts)-1], "/")
}

func makeFile(root, key string) string {
	return root + key
}

func checksum(fullpath string) (string, error) {
	file, err := os.Open(fullpath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	io.Copy(hash, file)
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
