package cache

import (
	"context"
	"crypto"
	"encoding/base64"
	"io"
	"path/filepath"
	"strings"

	"github.com/dokufy/paperfit/internal/encryption"
	"github.com/dokufy/paperfit/internal/logger"
	"github.com/dokufy/paperfit/internal/service/storage"
	"github.com/eko/gocache/lib/v4/store"
)

type Cache struct {
	cacheAdapter  *storage.CacheAdapter
	encryptionKey *string
	logger        *logger.Logger
}

type CacherOptions func(cache *Cache) error

func New(dsn string, options ...CacherOptions) (*Cache, error) {
	cache := &Cache{}

	if options != nil {
		for _, cfg := range options {
			if cfg != nil {
				err := cfg(cache)
				if err != nil {
					return &Cache{}, err
				}
			}
		}
	}

	storageOpts := []storage.StorageOption{}

	if cache.logger != nil {
		storageOpts = append(storageOpts, storage.WithLogger(*cache.logger))
	}

	store, err := storage.NewStorageAdapter(dsn, storageOpts...)
	if err != nil {
		return nil, err
	}

	cache.cacheAdapter = storage.NewCacheAdapter(store)

	return cache, nil
}

func WithEncryption(key string) CacherOptions {
	return func(cache *Cache) error {
		cache.encryptionKey = &key

		return nil
	}
}

func WithLogger(logger *logger.Logger) CacherOptions {
	return func(cache *Cache) error {
		cache.logger = logger

		return nil
	}
}

func (r *Cache) GetArchive(ctx context.Context, key string) (io.Reader, error) {
	return get(ctx, r, key, "archive")
}

func (r *Cache) GetPDF(ctx context.Context, key string) (io.Reader, error) {
	return get(ctx, r, key, "pdf")
}

func (r *Cache) SetArchive(ctx context.Context, key string, archive io.Reader, size int64) error {
	return set(ctx, r, key, archive, size, "archive")
}

func (r *Cache) SetPDF(ctx context.Context, key string, pdf io.Reader, size int64) error {
	return set(ctx, r, key, pdf, size, "pdf")
}

func (r *Cache) DeleteArchive(ctx context.Context, key string) error {
	return delete(ctx, r, key, "archive")
}

func (r *Cache) DeletePDF(ctx context.Context, key string) error {
	return delete(ctx, r, key, "pdf")
}

func Key(keys ...string) (string, error) {
	hash := crypto.SHA1.New()
	compositeKey := strings.Join(keys, "")

	_, err := hash.Write([]byte(compositeKey))
	if err != nil {
		return "", err
	}

	sha := hash.Sum(nil)

	return base64.URLEncoding.EncodeToString(sha), nil
}

func get(ctx context.Context, cache *Cache, key string, basePath string) (io.Reader, error) {
	path := filepath.Join(basePath, key)

	pdf, err := cache.cacheAdapter.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if pdf == nil {
		return nil, nil
	}

	if cache.encryptionKey != nil {
		r, err := encryption.DecryptingReader(*cache.encryptionKey, pdf.(io.Reader))
		if err != nil {
			return nil, err
		}

		return r, nil
	}

	return pdf.(io.Reader), nil
}

func set(ctx context.Context, cache *Cache, key string, pdf io.Reader, size int64, basePath string) error {
	path := filepath.Join(basePath, key)

	reader := pdf

	if cache.encryptionKey != nil {
		r, err := encryption.EncryptingReader(*cache.encryptionKey, pdf)
		if err != nil {
			return err
		}

		reader = r
	}

	err := cache.cacheAdapter.Set(ctx, path, reader, store.WithCost(size))
	if err != nil {
		return err
	}

	return nil
}

func delete(ctx context.Context, cache *Cache, key string, basePath string) error {
	path := filepath.Join(basePath, key)

	err := cache.cacheAdapter.Delete(ctx, path)
	if err != nil {
		return err
	}

	return nil
}
