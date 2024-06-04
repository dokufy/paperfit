package cacher

import (
	"bytes"
	"context"
	"crypto"
	"encoding/base64"
	"io"
	"path/filepath"

	"github.com/atoncetti/combiner/internal/cache"
	"github.com/atoncetti/combiner/internal/logger"
	"github.com/atoncetti/combiner/internal/storage"
	"github.com/eko/gocache/lib/v4/store"
)

type Cacher struct {
	cache *cache.Cache
}

func New(dsn *string, logger logger.Logger) (Cacher, error) {
	if dsn == nil {
		return Cacher{}, nil
	}

	storage, err := storage.New(*dsn, storage.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create storage", err)

		return Cacher{}, err
	}

	return Cacher{cache: cache.New(storage)}, nil
}

func (r *Cacher) GetArchive(ctx context.Context, key string) (io.Reader, error) {
	return get(ctx, r, key, "archive")
}

func (r *Cacher) GetPDF(ctx context.Context, key string) (io.Reader, error) {
	return get(ctx, r, key, "pdf")
}

func (r *Cacher) SetArchive(ctx context.Context, key string, archive io.Reader, size int64) error {
	return set(ctx, r, key, archive, size, "archive")
}

func (r *Cacher) SetPDF(ctx context.Context, key string, pdf io.Reader, size int64) error {
	return set(ctx, r, key, pdf, size, "pdf")
}

func (r *Cacher) DeleteArchive(ctx context.Context, key string) error {
	return delete(ctx, r, key, "archive")
}

func (r *Cacher) DeletePDF(ctx context.Context, key string) error {
	return delete(ctx, r, key, "pdf")
}

func Key(keys [][]byte) (string, error) {
	hash := crypto.SHA1.New()
	compositeKey := bytes.Join(keys, []byte{})

	_, err := hash.Write(compositeKey)
	if err != nil {
		return "", err
	}

	sha := hash.Sum(nil)

	return base64.URLEncoding.EncodeToString(sha), nil
}

func get(ctx context.Context, cacher *Cacher, key string, basePath string) (io.Reader, error) {
	path := filepath.Join(basePath, key)

	pdf, err := cacher.cache.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if pdf == nil {
		return nil, nil
	}

	return pdf.(io.Reader), nil
}

func set(ctx context.Context, cacher *Cacher, key string, pdf io.Reader, size int64, basePath string) error {
	path := filepath.Join(basePath, key)

	err := cacher.cache.Set(ctx, path, pdf, store.WithCost(size))
	if err != nil {
		return err
	}

	return nil
}

func delete(ctx context.Context, cacher *Cacher, key string, basePath string) error {
	path := filepath.Join(basePath, key)

	err := cacher.cache.Delete(ctx, path)
	if err != nil {
		return err
	}

	return nil
}
