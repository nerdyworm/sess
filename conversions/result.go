package conversions

import "io"

type Result struct {
	Key   string
	Error error
}

type Converter interface {
	Key() string
	ContentType() string
	Convert() (io.ReadCloser, error)
}
