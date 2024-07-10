package projectprovider

import (
	"context"
	"io"
	"net/http"
	"path"

	"github.com/atoncetti/combiner/internal/archive"
	"github.com/atoncetti/combiner/internal/cache"
	"github.com/atoncetti/combiner/internal/httptool"
	"github.com/atoncetti/combiner/internal/logger"
	"github.com/atoncetti/combiner/internal/service"
)

type ArchiveProjectServiceProvider interface {
	GetProject(ctx context.Context, url string, headers map[string][]string) (service.Project, error)
}

type ArchiveProjectProvider struct {
	cache  *cache.Cache
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

func WithCache(cache cache.Cache) ArchiveProjectProviderOptions {
	return func(storage *ArchiveProjectProvider) error {
		storage.cache = &cache

		return nil
	}
}

func (p *ArchiveProjectProvider) GetProject(ctx context.Context, url string, headers map[string][]string) (service.Project, error) {
	var archiveName string
	var rawArchive io.Reader

	cacheKey, err := cache.Key([][]byte{[]byte(url), httptool.FlattenHeaders(headers)})
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
			rawArchive = cachedArchive
		}
	}

	if rawArchive == nil {
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
		rawArchive = res.Body
	}

	project, err := archive.ExtractToProject(ctx, archiveName, rawArchive)
	if err != nil {
		return service.Project{}, err
	}

	if p.cache != nil {
		// TODO: run this in a separate goroutine
		file, err := archive.CompressArchiveProject(ctx, &project)
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
