package archive

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"time"

	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/service"
	"github.com/mholt/archiver/v4"
)

type archiveFile struct {
	io.ReadCloser
	fs.FileInfo
}

func (f archiveFile) Stat() (fs.FileInfo, error) { return f.FileInfo, nil }

type ProjectExtractedResponse struct {
	project service.Project
	err     error
}

type ProjectCompressedResponse struct {
	file service.File
	err  error
}

func ExtractToProject(ctx context.Context, filename string, content io.Reader) (service.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	respch := make(chan ProjectExtractedResponse)

	go func() {
		format, input, err := archiver.Identify(ctx, filename, content)
		if err != nil {
			respch <- ProjectExtractedResponse{project: service.Project{}, err: err}
		}

		fileList := []*service.File{}

		handler := func(ctx context.Context, f archiver.FileInfo) error {
			if f.IsDir() {
				return nil
			}

			c, err := f.Open()
			if err != nil {
				return err
			}
			defer c.Close()

			buf := new(bytes.Buffer)
			_, err = io.CopyN(buf, c, f.Size())
			if err != nil {
				return err
			}

			file := service.NewFile(f.Name(), buf, f.Size())

			fileList = append(fileList, &file)

			return nil
		}

		if ex, ok := format.(archiver.Extractor); ok {
			err = ex.Extract(context.Background(), input, handler)
			if err != nil {
				logger.Log.Error("Failed to extract archive to project", err)

				respch <- ProjectExtractedResponse{project: service.Project{}, err: err}
			}
		}

		respch <- ProjectExtractedResponse{project: service.NewProject(fileList), err: err}
	}()

	for {
		select {
		case <-ctx.Done():
			return service.Project{}, ErrTimeoutExtracting
		case resp := <-respch:
			return resp.project, resp.err
		}
	}
}

func CompressArchiveProject(ctx context.Context, project *service.Project) (service.File, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	respch := make(chan ProjectCompressedResponse)

	go func() {
		buf := new(bytes.Buffer)
		var files []archiver.FileInfo

		format := archiver.Archive{
			Compression: archiver.Brotli{},
			Archival:    archiver.Tar{},
		}

		for _, file := range project.GetFiles() {
			if file != nil {
				files = append(files, archiver.FileInfo{
					FileInfo:      file,
					NameInArchive: file.Name(),
					Open: func() (fs.File, error) {
						buf := new(bytes.Buffer)
						tee := io.TeeReader(file.Content, buf)

						file.Content = buf

						return archiveFile{ReadCloser: io.NopCloser(tee), FileInfo: file}, nil
					},
				})
			}
		}

		err := format.Archive(ctx, buf, files)
		if err != nil {
			logger.Log.Error("Failed to compress archive", err)

			respch <- ProjectCompressedResponse{file: service.File{}, err: err}
		}

		respch <- ProjectCompressedResponse{file: service.NewFile("compressed", buf, int64(buf.Len()))}
	}()

	for {
		select {
		case <-ctx.Done():
			return service.File{}, ErrTimeoutCompressing
		case resp := <-respch:
			return resp.file, resp.err
		}
	}
}
