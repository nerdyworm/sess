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
	"github.com/nerdyworm/sess/models"
	"github.com/nerdyworm/sess/repos"
	"github.com/nerdyworm/sess/storage"
	"github.com/nerdyworm/sess/util"
)

type Options struct {
	Size   int
	Brand  bool
	Format string
}

type InstanceToJPG struct {
	InstanceID string
	Options    Options
}

// XXX - need to add brand position to this...
func (i InstanceToJPG) Key() string {
	hash := md5.New()

	io.WriteString(hash, i.InstanceID)
	io.WriteString(hash, fmt.Sprintf("%d", i.Options.Size))
	io.WriteString(hash, fmt.Sprintf("%b", i.Options.Brand))

	return fmt.Sprintf("convertions/%x.jpg", hash.Sum(nil))
}

func (i InstanceToJPG) ContentType() string {
	return "image/jpg"
}

func (i InstanceToJPG) Convert() (io.ReadCloser, error) {
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

	err = dicom.ExtractFirst()
	if err != nil {
		return nil, err
	}

	frame := dicom.InstanceKey()

	if dicom.Modality == "DOC" {
		err = convertDocToImage(frame)
		if err != nil {
			return nil, err
		}
	}

	if i.Options.Size > 0 {
		err = resizeImage(frame, i.Options.Size)
		if err != nil {
			return nil, err
		}
	}

	if i.Options.Brand {
		account, err := repos.Accounts.FindByID(instance.AccountID)
		if err != nil {
			return nil, err
		}

		err = applyAccountBranding(frame, account)
		if err != nil {
			return nil, err
		}
	}

	return os.Open(frame)
}

func resizeImage(path string, size int) error {
	resize := fmt.Sprintf("%dx%d^", size, size)
	extent := fmt.Sprintf("%dx%d", size, size)

	convert := exec.Command("convert", path, "-thumbnail", resize, "-gravity", "center", "-extent", extent, "jpeg:"+path)
	output, err := convert.CombinedOutput()
	if err != nil {
		log.Printf("resizeImage stderr dump \n%s\n", string(output))
		return err
	}

	return nil
}

func convertDocToImage(path string) error {
	convert := exec.Command(
		"convert",
		path,
		path+"-%05d.jpg",
	)

	output, err := convert.CombinedOutput()
	if err != nil {
		log.Printf("convert doc to image dump \n%s\n", string(output))
		return err
	}

	err = os.Rename(path+"-00000.jpg", path)
	if err != nil {
		return err
	}

	return nil
}

func applyAccountBranding(path string, account *models.Account) error {
	r, err := storage.Primary.Get(account.LogoKey())
	if err != nil {
		return err
	}
	defer r.Close()

	k := util.RandomString(32)

	err = storage.Scratch.Put(k, r)
	if err != nil {
		return err
	}
	defer storage.Scratch.Delete(k)

	p := storage.Scratch.GetPath(k)

	composite := exec.Command(
		"composite",
		"-gravity",
		gravityForBrandingLogo(account),
		p,
		path,
		path,
	)

	output, err := composite.CombinedOutput()
	if err != nil {
		log.Printf("applyAccountBranding dump \n%s\n", string(output))
		return err
	}

	return nil
}

var (
	defaultGravity   = "NorthWest"
	settingToGravity = map[string]string{
		"top_left":     "NorthWest",
		"top_right":    "NorthEast",
		"bottom_left":  "SouthWest",
		"bottom_right": "SouthEast",
	}
)

func gravityForBrandingLogo(account *models.Account) string {
	if setting, ok := settingToGravity[account.Settings.LogoPosition]; ok {
		return setting
	}

	return defaultGravity
}
