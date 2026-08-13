package kafka

import (
    "context"
    "encoding/json"
    "log"
    "time"

    "github.com/segmentio/kafka-go"
    "order-service/internal/model"
    "order-service/internal/store"
    "order-service/internal/cache"
)

type Consumer struct {
    reader *kafka.Reader
    store  *store.Store
    cache  *cache.Cache
}

func NewConsumer(broker, topic, groupID string, st *store.Store, c *cache.Cache) *Consumer {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:        []string{broker},
        Topic:          topic,
        GroupID:        groupID,
        MinBytes:       10e3,
        MaxBytes:       10e6,
        CommitInterval: time.Second,
        StartOffset:    kafka.LastOffset,
    })
    return &Consumer{
        reader: reader,
        store:  st,
        cache:  c,
    }
}

func (c *Consumer) Start(ctx context.Context) {
    go func() {
        for {
            msg, err := c.reader.FetchMessage(ctx)
            if err != nil {
                if ctx.Err() != nil {
                    return
                }
                log.Printf("error fetching message: %v", err)
                continue
            }

            var order model.Order
            if err := json.Unmarshal(msg.Value, &order); err != nil {
                log.Printf("invalid message: %v, data: %s", err, string(msg.Value))
                // коммитим, чтобы не застрять на мусоре
                if err := c.reader.CommitMessages(ctx, msg); err != nil {
                    log.Printf("commit error: %v", err)
                }
                continue
            }

            // Сохраняем в БД
            if err := c.store.SaveOrder(ctx, order); err != nil {
                log.Printf("failed to save order %s: %v", order.OrderUID, err)
                // не коммитим, пробуем снова
                continue
            }

            // Обновляем кеш
            c.cache.Set(order.OrderUID, &order)

            // Подтверждаем смещение
            if err := c.reader.CommitMessages(ctx, msg); err != nil {
                log.Printf("commit error: %v", err)
            }
        }
    }()
}

func (c *Consumer) Close() error {
    return c.reader.Close()
}