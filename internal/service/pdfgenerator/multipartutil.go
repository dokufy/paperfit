package pdfgenerator

import (
	"bytes"
	"io"
	"mime/multipart"

	"github.com/atoncetti/combiner/internal/service"
)

type Multipart struct {
	buf    io.Reader
	writer *multipart.Writer
}

func MultipartForm(files []*service.File, fields *map[string]string) (*Multipart, error) {
	var (
		buf = new(bytes.Buffer)
		w   = multipart.NewWriter(buf)
	)

	err := addFilesToMultipart(w, files)
	if err != nil {
		return nil, err
	}

	if fields != nil {
		addFieldsToMultipart(w, fields)
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	return &Multipart{buf: buf, writer: w}, nil
}

func (m *Multipart) Body() io.Reader {
	return m.buf
}

func (m *Multipart) FormDataContentType() string {
	return m.writer.FormDataContentType()
}

func addFilesToMultipart(multipartWriter *multipart.Writer, files []*service.File) error {
	for _, file := range files {
		if file != nil {
			part, err := multipartWriter.CreateFormFile(file.Name(), file.Name())
			if err != nil {
				return err
			}

			buf := new(bytes.Buffer)

			_, err = io.CopyN(buf, file.Content, file.Size())
			if err != nil {
				return err
			}

			_, err = part.Write(buf.Bytes())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func addFieldsToMultipart(multipartWriter *multipart.Writer, fields *map[string]string) {
	for key, value := range *fields {
		if key != "" && value != "" {
			_ = multipartWriter.WriteField(key, value)
		}
	}
}
