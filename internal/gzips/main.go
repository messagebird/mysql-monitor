package gzips

import (
	"bytes"
	"compress/gzip"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/messagebird/mysql-monitor/internal/logging"
	"os"
)

// Gzipper is a wrapper around .gz
type Gzipper struct {
	gz *gzip.Writer
	file *os.File
	fileName string
}

// NewGzipper returns a wrapper around .gz
func NewGzipper(fileName string) (*Gzipper, error) {
	logging.Trace(logging.TraceTypeEntering)
	defer logging.Trace(logging.TraceTypeExiting)

	file, err := os.OpenFile(fileName, os.O_CREATE | os.O_APPEND| os.O_WRONLY, 0666)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create file %s", fileName)
	}
	gz := gzip.NewWriter(file)

	return &Gzipper{fileName: fileName, gz: gz, file: file}, nil
}

func (g *Gzipper) Close() error {
	logging.Trace(logging.TraceTypeEntering)
	defer logging.Trace(logging.TraceTypeExiting)

	err := g.gz.Close()
	if err != nil {
		return errors.Wrap(err, "could not close gz writer")
	}
	
	err = g.file.Close()
	if err != nil {
		return errors.Wrap(err, "could not close file")
	}
	
	return nil
}

func (g *Gzipper) Write(p []byte) (n int, err error) {
	logging.Trace(logging.TraceTypeEntering)
	defer logging.Trace(logging.TraceTypeExiting)

	return g.gz.Write(p)
}

func (g *Gzipper) Name() string {
	logging.Trace(logging.TraceTypeEntering)
	defer logging.Trace(logging.TraceTypeExiting)

	return g.fileName
}

func WriteToGzipper(buf *bytes.Buffer, filename string) error {
	logging.Trace(logging.TraceTypeEntering)
	defer logging.Trace(logging.TraceTypeExiting)

	gzipper, err := NewGzipper(filename)
	if err != nil {
		return errors.Wrap(err, "could not create gzipper")
	}

	defer func() {
		logging.Trace(logging.TraceTypeEntering)
		defer logging.Trace(logging.TraceTypeExiting)

		err := gzipper.Close()
		if err != nil {
			logrus.WithError(err).Error("could not close gzipper")
		}
	}()

	_, err = buf.WriteTo(gzipper)
	if err != nil {
		logrus.WithError(err).Error("could not write buffer to gzipper")
		return errors.Wrap(err, "could not write buffer to gzipper")
	}

	return nil
}