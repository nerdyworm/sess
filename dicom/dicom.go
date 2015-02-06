package dicom

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/nerdyworm/sess/util"

	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
)

var ROOT = "/tmp/scratch/dicom_root/"

type Dicom struct {
	Path              string
	SOPInstanceUID    string
	PatientID         string
	PatientName       string
	StudyInstanceUID  string
	SeriesInstanceUID string
	CineRate          string
	Modality          string
	NumberOfFrames    int
	WindowCenter      string
	WindowWidth       string
	Elements          []Element
	elementsByName    map[string]Element

	extractedFrames bool
	basePath        string
}

func New(path string) (dicom Dicom, err error) {
	dicom.basePath = util.RandomString(32)
	dicom.Path = path
	dicom.Elements = []Element{}
	dicom.elementsByName = make(map[string]Element)

	if err = dicom.ExtractAttributes(); err != nil {
		return
	}

	dicom.StudyInstanceUID = dicom.Get("StudyInstanceUID").Value
	dicom.SeriesInstanceUID = dicom.Get("SeriesInstanceUID").Value
	dicom.SOPInstanceUID = dicom.Get("SOPInstanceUID").Value
	dicom.PatientID = dicom.Get("PatientID").Value
	dicom.PatientName = dicom.Get("PatientName").Value
	dicom.CineRate = dicom.Get("CineRate").Value
	dicom.Modality = dicom.Get("Modality").Value
	dicom.WindowCenter = dicom.Get("WindowCenter").Value
	dicom.WindowWidth = dicom.Get("WindowWidth").Value

	frames := dicom.Get("NumberOfFrames").Value
	if frames != "" {
		dicom.NumberOfFrames, _ = strconv.Atoi(frames)
	}

	return
}

func (d Dicom) Get(name string) Element {
	return d.elementsByName[name]
}

func (d Dicom) IsCine() bool {
	return d.Get("CineRate").Value != ""
}

func (d *Dicom) Clean() error {
	return os.RemoveAll(d.fsRoot())
}

func (d *Dicom) fsRoot() string {
	return ROOT + d.basePath + "/"
}

func (d Dicom) StudyKey() string {
	return d.fsRoot() + d.StudyInstanceUID
}

func (d Dicom) SeriesKey() string {
	return d.StudyKey() + "/" + d.SeriesInstanceUID
}

func (d Dicom) InstanceKey() string {
	return d.SeriesKey() + "/" + d.SOPInstanceUID
}

func (d *Dicom) ExtractFirst() error {
	root := d.SeriesKey()
	if err := os.MkdirAll(root, 0777); err != nil {
		return err
	}

	if d.Modality == "CT" {
		dcmj2pnm := exec.Command("dcmj2pnm", "--conv-guess-lossy", "--write-jpeg", "+Ww", d.WindowCenter, d.WindowWidth, d.Path, d.InstanceKey())
		output, err := dcmj2pnm.CombinedOutput()
		if err != nil {
			log.Printf("dcmj2pnm error\n%s\n", string(output))
			return err
		}
	} else if d.Modality == "DOC" {
		dcm2pdf := exec.Command("dcm2pdf", d.Path, d.InstanceKey())
		output, err := dcm2pdf.CombinedOutput()
		if err != nil {
			log.Printf("dcm2pdf error\n%s\n", string(output))
			return err
		}
	} else {
		dcmj2pnm := exec.Command("dcmj2pnm", "--conv-guess-lossy", "--write-jpeg", d.Path, d.InstanceKey())
		output, err := dcmj2pnm.CombinedOutput()
		if err != nil {
			log.Printf("dcmj2pnm error\n%s\n", string(output))
			return err
		}
	}

	return nil
}

func (d *Dicom) Extract() error {
	if d.extractedFrames {
		return nil
	}

	root := d.SeriesKey()
	if err := os.MkdirAll(root, 0777); err != nil {
		return err
	}

	if d.Modality == "CT" {
		dcmj2pnm := exec.Command("dcmj2pnm", "--all-frames", "--write-jpeg", "+Ww", d.WindowCenter, d.WindowWidth, d.Path, d.InstanceKey())
		output, err := dcmj2pnm.CombinedOutput()
		if err != nil {
			log.Printf("dcmj2pnm error\n%s\n", string(output))
			return err
		}
	} else if d.Modality == "DOC" {
		dcm2pdf := exec.Command("dcm2pdf", d.Path, d.InstanceKey())
		output, err := dcm2pdf.CombinedOutput()
		if err != nil {
			log.Printf("dcm2pdf error\n%s\n", string(output))
			return err
		}
	} else {
		var w sync.WaitGroup

		batchSize := d.NumberOfFrames / 4
		for i := 1; i <= d.NumberOfFrames; i += batchSize {
			w.Add(1)

			go func(start int) {
				dcmj2pnm := exec.Command(
					"dcmj2pnm",
					"--write-jpeg",
					"--conv-guess-lossy",
					"--use-frame-number",
					"--frame-range",
					fmt.Sprintf("%d", start),
					fmt.Sprintf("%d", batchSize),
					d.Path,
					d.InstanceKey(),
				)

				output, err := dcmj2pnm.CombinedOutput()
				if err != nil {
					log.Printf("dcmj2pnm error\n%s\n", string(output))
				}

				w.Done()
			}(i)
		}

		w.Wait()

		for i := 1; i <= d.NumberOfFrames; i++ {
			oldFilename := fmt.Sprintf("%s.f%d.jpg", d.InstanceKey(), i)
			newFilename := fmt.Sprintf("%s.%05d.jpg", d.InstanceKey(), i)
			err := os.Rename(oldFilename, newFilename)
			if err != nil {
				return err
			}
		}
	}

	d.extractedFrames = true
	return nil
}

func (d *Dicom) ExtractAttributes() error {
	dcm2xml := exec.Command("dcm2xml", d.Path)

	output, err := dcm2xml.CombinedOutput()
	if err != nil {
		log.Printf("dcm2xml error\n%s\n", string(output))
		return err
	}

	dcm, err := dcm2xmlDecode(output)
	if err != nil {
		log.Printf("decoding dcm2xmlOutput `%v`\n", err)
		return err
	}

	for _, element := range dcm.MetaHeader.Elements {
		d.elementsByName[element.Name] = element
		d.Elements = append(d.Elements, element)
	}

	for _, element := range dcm.DataSet.Elements {
		d.elementsByName[element.Name] = element
		d.Elements = append(d.Elements, element)
	}

	return nil
}

func dcm2xmlDecode(output []byte) (dcm dcm2xmlOutput, err error) {
	decoder := xml.NewDecoder(bytes.NewReader(output))
	decoder.CharsetReader = func(c string, input io.Reader) (io.Reader, error) {
		return charset.NewReader(c, input)
	}

	return dcm, decoder.Decode(&dcm)
}

type dcm2xmlOutput struct {
	MetaHeader struct {
		Elements []Element `xml:"element"`
	} `xml:"meta-header"`

	DataSet struct {
		Sequences []Sequence `xml:"sequence"`
		Elements  []Element  `xml:"element"`
	} `xml:"data-set"`
}

type Sequence struct {
	Card  int    `xml:"card,attr"`
	Name  string `xml:"name,attr"`
	Tag   string `xml:"tag,attr"`
	Vr    int    `xml:"vr,attr"`
	Items []struct {
		Elements []Element `xml:"element"`
	} `xml:"item"`
}

type Element struct {
	Name  string `xml:"name,attr"`
	Len   int    `xml:"len,attr"`
	Vm    int    `xml:"vm,attr"`
	Vr    int    `xml:"vr,attr"`
	Tag   string `xml:"tag,attr"`
	Value string `xml:",chardata"`
}
