package cache

import (
	"context"

	"github.com/eko/gocache/lib/v4/store"
)

const CacheType = "default"

type Cache struct {
	storage store.StoreInterface
}

func New(store store.StoreInterface) *Cache {
	return &Cache{
		storage: store,
	}
}

func (c *Cache) Get(ctx context.Context, key string) (any, error) {
	return c.storage.Get(ctx, key)
}

func (c *Cache) Set(ctx context.Context, key string, object any, options ...store.Option) error {
	return c.storage.Set(ctx, key, object, options...)
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.storage.Delete(ctx, key)
}

func (c *Cache) Invalidate(ctx context.Context, options ...store.InvalidateOption) error {
	return c.storage.Invalidate(ctx, nil)
}

func (c *Cache) Clear(ctx context.Context) error {
	return c.storage.Clear(ctx)
}

func (c *Cache) GetType() string {
	return CacheType
}
