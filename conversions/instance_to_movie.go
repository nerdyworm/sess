package conversions

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/nerdyworm/sess/dicom"
	"github.com/nerdyworm/sess/repos"
	"github.com/nerdyworm/sess/storage"
	"github.com/nerdyworm/sess/util"
)

type InstanceToMovie struct {
	InstanceID string
	Options    Options
}

func (i InstanceToMovie) Key() string {
	hash := md5.New()

	io.WriteString(hash, i.InstanceID)
	io.WriteString(hash, fmt.Sprintf("%d", i.Options.Size))
	io.WriteString(hash, fmt.Sprintf("%b", i.Options.Brand))

	return fmt.Sprintf("convertions/%x.mp4", hash.Sum(nil))
}

func (i InstanceToMovie) ContentType() string {
	return "video/mp4"
}

func (i InstanceToMovie) Convert() (io.ReadCloser, error) {
	if i.InstanceID == "" {
		return nil, errors.New("Empty Mongo ID")
	}

	instance, err := repos.Instances.FindByID(i.InstanceID)
	if err != nil {
		return nil, err
	}

	reader, err := storage.Primary.Get(instance.Key())
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	key := util.RandomString(32) + instance.Key()

	err = storage.Scratch.Put(key, reader)
	if err != nil {
		return nil, err
	}
	defer storage.Scratch.Delete(key)

	path := storage.Scratch.GetPath(key)

	dicom, err := dicom.New(path)
	if err != nil {
		return nil, err
	}
	defer dicom.Clean()

	err = dicom.Extract()
	if err != nil {
		return nil, err
	}

	frame := dicom.InstanceKey()
	movieFilename := frame + ".mp4"

	patern := dicom.InstanceKey() + ".%05d.jpg"

	rate := dicom.CineRate
	if rate == "" {
		rate = "1"
	}

	convert := exec.Command(
		"ffmpeg",
		"-y",
		"-r", rate,
		"-i", patern,
		"-c:v", "libx264",
		"-r", rate,
		"-pix_fmt", "yuv420p",
		movieFilename,
	)

	output, err := convert.CombinedOutput()
	if err != nil {
		log.Printf("%s\n", string(output))
		return nil, err
	}

	return os.Open(movieFilename)
}
