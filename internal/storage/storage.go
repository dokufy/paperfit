package storage

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/atoncetti/combiner/internal/logger"
	"github.com/eko/gocache/lib/v4/store"
	_ "go.beyondstorage.io/services/fs/v4"
	_ "go.beyondstorage.io/services/minio"
	_ "go.beyondstorage.io/services/s3/v3"
	"go.beyondstorage.io/v5/services"
	"go.beyondstorage.io/v5/types"
)

type Storage struct {
	store  types.Storager
	logger *logger.Logger
}

type StorageConfiguration func(storage *Storage) error

func New(url string, configs ...StorageConfiguration) (*Storage, error) {
	store, err := services.NewStoragerFromString(url)
	if err != nil {
		return nil, err
	}

	storage := &Storage{store: store}

	for _, cfg := range configs {
		err := cfg(storage)
		if err != nil {
			return nil, err
		}
	}

	return storage, nil
}

func WithLogger(logger logger.Logger) StorageConfiguration {
	return func(storage *Storage) error {
		storage.logger = &logger

		return nil
	}
}

func (s *Storage) GetWithTTL(ctx context.Context, key any) (any, time.Duration, error) {
	value, err := s.Get(ctx, key)

	return value, 0, err
}

func (s *Storage) Get(ctx context.Context, key any) (any, error) {
	var buf bytes.Buffer

	_, err := s.store.ReadWithContext(ctx, key.(string), &buf)
	if err != nil {
		s.logger.Debug("failed to read key", key, err)
		return nil, nil
	}

	return &buf, nil
}

func (s *Storage) Set(ctx context.Context, key any, value any, options ...store.Option) error {
	opts := store.ApplyOptions(options...)

	_, err := s.store.WriteWithContext(ctx, key.(string), value.(io.Reader), opts.Cost)
	if err != nil {
		s.logger.Error("failed to write key: %v", key, err)
		return err
	}

	return nil
}

func (s *Storage) Delete(ctx context.Context, key any) error {
	return s.store.DeleteWithContext(ctx, key.(string))
}

func (s *Storage) Invalidate(ctx context.Context, options ...store.InvalidateOption) error {
	return nil
}

func (s *Storage) Clear(ctx context.Context) error {
	return nil
}

func (s *Storage) GetType() string {
	return s.store.Metadata().Name
}
