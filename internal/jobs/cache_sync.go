package jobs

import (
	"github.com/emrgen/tinydoc/internal/cache"
	"gorm.io/gorm"
)

type CacheSyncTask struct {
	cache cache.DocumentCache
	db    *gorm.DB
	cron  string
}

func NewCacheSyncTask(interval string, db *gorm.DB, cache cache.DocumentCache) *CacheSyncTask {
	return &CacheSyncTask{
		db:    db,
		cache: cache,
		cron:  interval,
	}
}

func (c *CacheSyncTask) ID() string {
	return "cache_sync"
}

func (c *CacheSyncTask) Name() string {
	return "cache_sync"
}

func (c *CacheSyncTask) Schedule() string {
	return c.cron
}

func (c *CacheSyncTask) Run() {
	// err := c.cache.Sync(context.Background(), 100, c.db)

	// if err != nil {
	// 	return
	// }
}
