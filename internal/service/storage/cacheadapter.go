package storage

import (
	"context"

	"github.com/eko/gocache/lib/v4/store"
)

const CacheType = "default"

type CacheAdapter struct {
	storage store.StoreInterface
}

func NewCacheAdapter(store store.StoreInterface) *CacheAdapter {
	return &CacheAdapter{
		storage: store,
	}
}

func (c *CacheAdapter) Get(ctx context.Context, key string) (any, error) {
	return c.storage.Get(ctx, key)
}

func (c *CacheAdapter) Set(ctx context.Context, key string, object any, options ...store.Option) error {
	return c.storage.Set(ctx, key, object, options...)
}

func (c *CacheAdapter) Delete(ctx context.Context, key string) error {
	return c.storage.Delete(ctx, key)
}

func (c *CacheAdapter) Invalidate(ctx context.Context, options ...store.InvalidateOption) error {
	return c.storage.Invalidate(ctx, nil)
}

func (c *CacheAdapter) Clear(ctx context.Context) error {
	return c.storage.Clear(ctx)
}

func (c *CacheAdapter) GetType() string {
	return CacheType
}
