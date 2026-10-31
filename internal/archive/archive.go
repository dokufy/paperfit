package archive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"
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

var (
	htmlAttrRe = regexp.MustCompile(`(?i)(src|href)\s*=\s*("([^"]*)"|'([^']*)')`)
	cssUrlRe   = regexp.MustCompile(`(?i)url\(\s*(['"]?)([^'")]+)['"]?\s*\)`)
)

func ExtractToProject(ctx context.Context, filename string, content io.Reader) (service.Project, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	type result struct {
		project service.Project
		err     error
	}
	resCh := make(chan result, 1)

	go func() {
		format, input, err := archiver.Identify(ctx, filename, content)
		if err != nil {
			resCh <- result{project: service.Project{}, err: err}
			return
		}

		fileList := []*service.File{}
		// store raw contents for rewriting
		type extracted struct {
			orig    string
			name    string
			content []byte
			size    int64
		}
		var extractedFiles []extracted

		handler := func(ctx context.Context, f archiver.FileInfo) error {
			if f.IsDir() {
				return nil
			}

			c, err := f.Open()
			if err != nil {
				return err
			}
			defer func() { _ = c.Close() }()

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, c)
			if err != nil {
				return err
			}

			origName := f.NameInArchive
			if origName == "" {
				origName = f.Name()
			}

			extractedFiles = append(extractedFiles, extracted{
				orig:    origName,
				name:    filepath.Base(origName),
				content: buf.Bytes(),
				size:    int64(buf.Len()),
			})

			return nil
		}

		if ex, ok := format.(archiver.Extractor); ok {
			err = ex.Extract(ctx, input, handler)
			if err != nil {
				logger.Log.Error("Failed to extract archive to project", err)

				resCh <- result{project: service.Project{}, err: err}
				return
			}
		}

		// Assign unique flattened names for colliding basenames.
		used := map[string]bool{}
		origToFlat := map[string]string{}
		for i := range extractedFiles {
			base := filepath.Base(extractedFiles[i].orig)
			if base == "" || base == "." || base == "/" {
				base = extractedFiles[i].orig
			}
			flat := base
			if used[flat] {
				ext := filepath.Ext(base)
				stem := strings.TrimSuffix(base, ext)
				n := 1
				for {
					cand := fmt.Sprintf("%s_%d%s", stem, n, ext)
					if !used[cand] {
						flat = cand
						break
					}
					n++
				}
			}
			used[flat] = true
			extractedFiles[i].name = flat
			origToFlat[extractedFiles[i].orig] = flat
		}

		// Rewrite references in index.html and css files.
		for i := range extractedFiles {
			name := extractedFiles[i].name
			lower := strings.ToLower(name)
			if lower == "index.html" || strings.HasSuffix(lower, ".css") || strings.HasSuffix(lower, ".htm") || strings.HasSuffix(lower, ".html") {
				text := string(extractedFiles[i].content)
				rewritten := rewriteReferences(text, origToFlat)
				extractedFiles[i].content = []byte(rewritten)
				extractedFiles[i].size = int64(len(rewritten))
			}
		}

		for _, ef := range extractedFiles {
			buf := bytes.NewReader(ef.content)
			file := service.NewFile(ef.name, buf, ef.size)
			filePtr := file
			fileList = append(fileList, &filePtr)
		}

		resCh <- result{project: service.NewProject(fileList), err: nil}
	}()

	select {
	case <-ctx.Done():
		return service.Project{}, ErrTimeoutExtracting
	case r := <-resCh:
		return r.project, r.err
	}
}

func resolveFlat(val string, nameMap map[string]string) (string, bool) {
	clean := strings.TrimPrefix(strings.TrimSpace(val), "./")
	if flat, ok := nameMap[clean]; ok {
		return flat, true
	}
	base := path.Base(clean)
	var matches []string
	for orig, flat := range nameMap {
		if path.Base(orig) == base {
			matches = append(matches, flat)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	return "", false
}

func rewriteReferences(content string, nameMap map[string]string) string {
	// HTML attributes
	content = htmlAttrRe.ReplaceAllStringFunc(content, func(match string) string {
		loc := htmlAttrRe.FindStringSubmatch(match)
		if len(loc) < 4 {
			return match
		}
		var val string
		if loc[3] != "" {
			val = loc[3]
		} else {
			val = loc[4]
		}
		quote := "\""
		if strings.Contains(match, "'"+val+"'") {
			quote = "'"
		}
		flat, ok := resolveFlat(val, nameMap)
		if ok && flat != "" && flat != val {
			attr := loc[1]
			return attr + "=" + quote + flat + quote
		}
		if ok && flat != "" && flat == val {
			return match
		}
		base := path.Base(strings.TrimSpace(val))
		base = strings.TrimPrefix(base, "./")
		if base != "" {
			if f, ok2 := nameMap[base]; ok2 && f != val {
				attr := loc[1]
				return attr + "=" + quote + f + quote
			}
		}
		return match
	})

	// CSS url()
	content = cssUrlRe.ReplaceAllStringFunc(content, func(match string) string {
		loc := cssUrlRe.FindStringSubmatch(match)
		if len(loc) < 3 {
			return match
		}
		quote := loc[1]
		val := loc[2]
		flat, ok := resolveFlat(val, nameMap)
		if ok && flat != "" {
			return "url(" + quote + flat + quote + ")"
		}
		return match
	})

	return content
}

func CompressArchiveProject(ctx context.Context, project *service.Project) (service.File, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	type result struct {
		file service.File
		err  error
	}
	resCh := make(chan result, 1)

	go func() {
		buf := new(bytes.Buffer)
		var files []archiver.FileInfo

		format := archiver.Archive{
			Compression: archiver.Brotli{},
			Archival:    archiver.Tar{},
		}

		for _, file := range project.GetFiles() {
			if file != nil {
				data, _ := io.ReadAll(file.Content)
				file.Content = bytes.NewReader(data)
				dataCopy := append([]byte(nil), data...)
				files = append(files, archiver.FileInfo{
					FileInfo:      file,
					NameInArchive: file.Name(),
					Open: func() (fs.File, error) {
						return archiveFile{ReadCloser: io.NopCloser(bytes.NewReader(dataCopy)), FileInfo: file}, nil
					},
				})
			}
		}

		err := format.Archive(ctx, buf, files)
		if err != nil {
			logger.Log.Error("Failed to compress archive", err)

			resCh <- result{file: service.File{}, err: err}
			return
		}

		resCh <- result{file: service.NewFile("compressed", buf, int64(buf.Len()))}
	}()

	select {
	case <-ctx.Done():
		return service.File{}, ErrTimeoutCompressing
	case r := <-resCh:
		return r.file, r.err
	}
}
