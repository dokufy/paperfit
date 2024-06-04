package projectprovider

import (
	"context"
	"io"
	"net/http"
	"path"

	"github.com/atoncetti/combiner/internal/archiver"
	"github.com/atoncetti/combiner/internal/cacher"
	"github.com/atoncetti/combiner/internal/httptool"
	"github.com/atoncetti/combiner/internal/logger"
	"github.com/atoncetti/combiner/internal/service"
)

type ArchiveProjectServiceProvider interface {
	GetProject(ctx context.Context, url string, headers map[string][]string) (service.Project, error)
}

type ArchiveProjectProvider struct {
	cache  *cacher.Cacher
	logger *logger.Logger
}

type ArchiveProjectProviderOptions func(storage *ArchiveProjectProvider) error

func NewArchiveProjectProvider(logger *logger.Logger, options ...ArchiveProjectProviderOptions) (*ArchiveProjectProvider, error) {
	provider := &ArchiveProjectProvider{logger: logger}

	for _, cfg := range options {
		err := cfg(provider)
		if err != nil {
			return &ArchiveProjectProvider{}, err
		}
	}

	return provider, nil
}

func WithCache(cache cacher.Cacher) ArchiveProjectProviderOptions {
	return func(storage *ArchiveProjectProvider) error {
		storage.cache = &cache

		return nil
	}
}

func (p *ArchiveProjectProvider) GetProject(ctx context.Context, url string, headers map[string][]string) (service.Project, error) {
	var archiveName string
	var archive io.Reader

	cacheKey, err := cacher.Key([][]byte{[]byte(url), httptool.FlattenHeaders(headers)})
	if err != nil {
		return service.Project{}, err
	}

	if p.cache != nil {
		cachedArchive, err := p.cache.GetArchive(ctx, cacheKey)
		if err != nil {
			return service.Project{}, err
		}

		if cachedArchive != nil {
			archiveName = "archive.tar.br"
			archive = cachedArchive
		}
	}

	if archive == nil {
		client := http.Client{}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return service.Project{}, err
		}

		req.Header = headers

		res, err := client.Do(req)
		if err != nil {
			return service.Project{}, err
		}

		archiveName = path.Base(url)
		archive = res.Body
	}

	project, err := archiver.ExtractToProject(ctx, archiveName, archive)
	if err != nil {
		return service.Project{}, err
	}

	if p.cache != nil {
		// TODO: run this in a separate goroutine
		file, err := archiver.CompressArchiveProject(ctx, &project)
		if err != nil {
			return service.Project{}, err
		}

		err = p.cache.SetArchive(ctx, cacheKey, file.Content, file.Size())
		if err != nil {
			return service.Project{}, err
		}
	}

	return project, nil
}
