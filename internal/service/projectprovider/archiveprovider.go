package projectprovider

import (
	"context"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/dokufy/paperfit/internal/archive"
	"github.com/dokufy/paperfit/internal/cache"
	"github.com/dokufy/paperfit/internal/httptool"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/service"
)

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

func (p *ArchiveProjectProvider) GetProject(ctx context.Context, url string, headers *map[string]string, rootPath string) (service.Project, error) {
	var archiveName string
	var rawArchive io.Reader

	cacheKey, err := cache.Key(url, httptool.FlattenHeaders(headers))
	if err != nil {
		return service.Project{}, err
	}

	if p.cache != nil {
		cachedArchive, err := p.cache.Get(ctx, cacheKey, rootPath)
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

		req.Header = buildHeader(headers)

		res, err := client.Do(req)
		if err != nil {
			return service.Project{}, err
		}

		if res.StatusCode == http.StatusNotFound {
			return service.Project{}, &RequestError{StatusCode: 404, Message: "archive not found"}
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

		err = p.cache.Set(ctx, cacheKey, file.Content, file.Size(), rootPath)
		if err != nil {
			return service.Project{}, err
		}
	}

	return project, nil
}

func buildHeader(headers *map[string]string) map[string][]string {
	r := make(map[string][]string)

	if headers == nil {
		return r
	}

	for k, v := range *headers {
		r[k] = strings.Split(v, ",")
	}

	return r
}
