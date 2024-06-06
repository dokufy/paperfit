package service

import (
	"io"
	"io/fs"
	"os"
	"time"
)

type File struct {
	name    string
	size    int64
	Content io.Reader
}

func NewFile(name string, content io.Reader, size int64) File {
	return File{
		name:    name,
		size:    size,
		Content: content,
	}
}

func (f *File) Name() string {
	return f.name
}

// Size returns the size of the file in bytes
func (f *File) Size() int64 {
	return f.size
}

// Mode returns the file mode bits
func (f *File) Mode() os.FileMode {
	return fs.FileMode(4)
}

// ModTime returns the modification time of the file
func (f *File) ModTime() time.Time {
	return time.Now()
}

// IsDir returns true if the file is a directory
func (f *File) IsDir() bool {
	return false
}

// Sys returns underlying data source (can return nil)
func (f *File) Sys() any {
	return nil
}
