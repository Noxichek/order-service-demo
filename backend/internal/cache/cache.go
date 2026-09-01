package cache

import (
    "sync"
    "time"
    "order-service/internal/model"
)

type Cache struct {
    mu         sync.RWMutex
    items      map[string]cacheEntry
    maxEntries int
    ttl        time.Duration
}

type cacheEntry struct {
    order   *model.Order
    addedAt time.Time
}

func New(maxEntries int, ttl time.Duration) *Cache {
    return &Cache{
        items:      make(map[string]cacheEntry),
        maxEntries: maxEntries,
        ttl:        ttl,
    }
}

func (c *Cache) Get(uid string) (*model.Order, bool) {
    c.mu.RLock()
    entry, ok := c.items[uid]
    c.mu.RUnlock()
    if !ok {
        return nil, false
    }
    if time.Since(entry.addedAt) > c.ttl {
        c.mu.Lock()
        delete(c.items, uid)
        c.mu.Unlock()
        return nil, false
    }
    return entry.order, true
}

func (c *Cache) Set(uid string, order *model.Order) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if len(c.items) >= c.maxEntries {
        // Удаляем случайную запись (или можно реализовать LRU)
        for k := range c.items {
            delete(c.items, k)
            break
        }
    }
    c.items[uid] = cacheEntry{order: order, addedAt: time.Now()}
}

// LoadFromDB теперь принимает функцию getAll и заполняет кэш
func (c *Cache) LoadFromDB(getAll func() ([]*model.Order, error)) error {
    orders, err := getAll()
    if err != nil {
        return err
    }
    c.mu.Lock()
    defer c.mu.Unlock()
    for _, o := range orders {
        if len(c.items) >= c.maxEntries {
            break
        }
        c.items[o.OrderUID] = cacheEntry{order: o, addedAt: time.Now()}
    }
    return nil
}
