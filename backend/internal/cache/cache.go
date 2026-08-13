package cache

import (
    "sync"
    "order-service/internal/model"
)

type Cache struct {
    mu    sync.RWMutex
    items map[string]*model.Order
}

func New() *Cache {
    return &Cache{
        items: make(map[string]*model.Order),
    }
}

func (c *Cache) Get(uid string) (*model.Order, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    order, ok := c.items[uid]
    return order, ok
}

func (c *Cache) Set(uid string, order *model.Order) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[uid] = order
}

// LoadFromDB загружает все заказы из базы в кэш при старте.
// Сигнатура: передаём store, который умеет получать список UID или все заказы.
func (c *Cache) LoadFromDB(getAll func() ([]*model.Order, error)) error {
    orders, err := getAll()
    if err != nil {
        return err
    }
    c.mu.Lock()
    defer c.mu.Unlock()
    for _, o := range orders {
        c.items[o.OrderUID] = o
    }
    return nil
}