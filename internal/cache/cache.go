package cache

import (
	"context"
	"crypto"
	"encoding/base64"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/dokufy/paperfit/internal/encryption"
	"github.com/dokufy/paperfit/internal/service/storage"
	"github.com/eko/gocache/lib/v4/store"
)

type Cache struct {
	cacheAdapter  *storage.CacheAdapter
	encryptionKey *string
}

type CacherOptions func(cache *Cache) error

func New(dsn string, options ...CacherOptions) (*Cache, error) {
	cache := &Cache{}

	for _, cfg := range options {
		if cfg != nil {
			err := cfg(cache)
			if err != nil {
				return &Cache{}, err
			}
		}
	}

	storageOpts := []storage.StorageOption{}

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

func (r *Cache) Get(ctx context.Context, key string, rootPath string) (io.Reader, error) {
	if err := ValidateRootPath(rootPath); err != nil {
		return nil, err
	}
	path := filepath.Join(strings.TrimPrefix(rootPath, "/"), key)

	pdf, err := r.cacheAdapter.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if pdf == nil {
		return nil, nil
	}

	if r.encryptionKey != nil {
		r, err := encryption.DecryptingReader(*r.encryptionKey, pdf.(io.Reader))
		if err != nil {
			return nil, err
		}

		return r, nil
	}

	return pdf.(io.Reader), nil
}

func (r *Cache) Set(ctx context.Context, key string, resource io.Reader, size int64, rootPath string) error {
	if err := ValidateRootPath(rootPath); err != nil {
		return err
	}
	path := filepath.Join(strings.TrimPrefix(rootPath, "/"), key)

	reader := resource

	if r.encryptionKey != nil {
		r, err := encryption.EncryptingReader(*r.encryptionKey, resource)
		if err != nil {
			return err
		}

		reader = r
	}

	err := r.cacheAdapter.Set(ctx, path, reader, store.WithCost(size))
	if err != nil {
		return err
	}

	return nil
}

func (r *Cache) Delete(ctx context.Context, key string, rootPath string) error {
	if err := ValidateRootPath(rootPath); err != nil {
		return err
	}
	path := filepath.Join(strings.TrimPrefix(rootPath, "/"), key)

	err := r.cacheAdapter.Delete(ctx, path)
	if err != nil {
		return err
	}

	return nil
}

func ValidateRootPath(rootPath string) error {
	for _, seg := range strings.Split(filepath.ToSlash(rootPath), "/") {
		if seg == ".." {
			return errors.New("invalid root path: contains '..' traversal component")
		}
	}
	trimmed := strings.TrimPrefix(rootPath, "/")
	if filepath.IsAbs(trimmed) {
		return errors.New("invalid root path: absolute path not allowed")
	}
	if strings.HasPrefix(filepath.Clean(rootPath), "..") {
		return errors.New("invalid root path: contains '..' traversal component")
	}
	return nil
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
