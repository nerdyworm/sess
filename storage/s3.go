package storage

import (
	"io"
	"log"
	"os"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/nerdyworm/sess/util"
)

type S3Store struct {
	auth   aws.Auth
	sss    *s3.S3
	bucket *s3.Bucket
}

func NewS3Store(bucketName string) S3Store {
	auth := aws.Auth{os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""}
	sss := s3.New(auth, aws.USEast)
	bucket := sss.Bucket(bucketName)
	return S3Store{auth, sss, bucket}
}

func (s S3Store) Get(key string) (io.ReadCloser, error) {
	log.Printf("S3Store#Get `%s`\n", key)
	return s.bucket.GetReader(key)
}

func (s S3Store) Put(key string, reader io.Reader) error {
	log.Printf("S3Store#Put `%s`\n", key)
	tmp := NewFileStore("/tmp/s3_store/puts/")

	tmpKey := key + util.RandomString(32)

	err := tmp.Put(tmpKey, reader)
	if err != nil {
		log.Printf("[ERROR] putting into s3 temp store `%v`", err)
		return err
	}

	file, err := tmp.Get(tmpKey)
	if err != nil {
		log.Printf("[ERROR] error opening `%v`", err)
		return err
	}
	defer file.Close()

	stat, err := file.(*os.File).Stat()
	if err != nil {
		log.Printf("[ERROR] file.Stat() `%v`", err)
		return err
	}

	size := stat.Size()
	contentType := ""
	err = s.bucket.PutReader(key, file, size, contentType, s3.BucketOwnerFull)

	if err != nil {
		log.Printf("[ERROR] PutReader `%v`", err)
		return err
	}

	return tmp.Delete(tmpKey)
}

func (s S3Store) Exists(key string) (bool, error) {
	log.Printf("S3Store#Exists `%s`\n", key)

	response, err := s.bucket.GetResponse(key)
	if err != nil {
		if err.Error() == "The specified key does not exist." {
			return false, nil
		} else {
			return false, err
		}
	}

	return response.StatusCode == 200, err
}

func (s S3Store) Delete(key string) error {
	return s.bucket.Del(key)
}
