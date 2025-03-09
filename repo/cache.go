package repo

import (
	"context"
	"fmt"
	"github.com/patrickmn/go-cache"
	"math/rand"
	"time"
)

type BaseCache interface {
	Get(ctx context.Context, prefix string, tenantID uint64, uniqKey interface{}) (interface{}, bool)
	Set(ctx context.Context, prefix string, tenantID uint64, uniqKey, value interface{})
	Del(ctx context.Context, prefix string, tenantID uint64, uniqKey interface{})
	Flush(ctx context.Context)
	Close(ctx context.Context) error
}

type baseCache struct {
	cache *cache.Cache
}

var defaultExpiration = time.Duration(rand.Intn(10))*time.Minute + 30*time.Minute

func NewBaseCache(_ context.Context) BaseCache {
	return &baseCache{
		cache: cache.New(30*time.Minute, 15*time.Minute),
	}
}

func (bc *baseCache) Get(_ context.Context, prefix string, tenantID uint64, uniqKey interface{}) (interface{}, bool) {
	return bc.cache.Get(bc.getKey(prefix, tenantID, uniqKey))
}

func (bc *baseCache) Set(_ context.Context, prefix string, tenantID uint64, uniqKey, value interface{}) {
	bc.cache.Set(bc.getKey(prefix, tenantID, uniqKey), value, defaultExpiration)
}

func (bc *baseCache) Del(_ context.Context, prefix string, tenantID uint64, uniqKey interface{}) {
	bc.cache.Delete(bc.getKey(prefix, tenantID, uniqKey))
}

func (bc *baseCache) getKey(prefix string, tenantID uint64, uniqKey interface{}) string {
	return fmt.Sprintf("%s:%d:%v", prefix, tenantID, uniqKey)
}

func (bc *baseCache) Flush(_ context.Context) {
	bc.cache.Flush()
}

func (bc *baseCache) Close(ctx context.Context) error {
	bc.Flush(ctx)
	return nil
}
