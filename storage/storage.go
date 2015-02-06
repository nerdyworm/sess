package storage

import (
	"io"
)

var (
	Primary = NewS3Store("ben-trice-space-development")
	//Cache   = NewS3Store("ben-trice-space-development")
	Cache   = NewFileStore("/tmp/scratch/cache/")
	Scratch = NewFileStore("/tmp/scratch/")
)

type Storage interface {
	Get(string) (io.ReadCloser, error)
	Put(string, io.Reader) error
	Exists(string) (bool, error)
	Delete(string) error
}
