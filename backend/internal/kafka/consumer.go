package kafka

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "time"

    "github.com/segmentio/kafka-go"
    "order-service/internal/cache"
    "order-service/internal/model"
    "order-service/internal/store"
)

const (
    maxRetries = 5
    baseDelay  = 500 * time.Millisecond
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
        CommitInterval: time.Second, // будем коммитить вручную
        StartOffset:    kafka.FirstOffset, // чтобы новые группы читали с начала
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
                time.Sleep(time.Second)
                continue
            }

            log.Printf("received message: topic=%s partition=%d offset=%d size=%d",
                msg.Topic, msg.Partition, msg.Offset, len(msg.Value))

            // 1. Проверка JSON
            var order model.Order
            if err := json.Unmarshal(msg.Value, &order); err != nil {
                log.Printf("invalid JSON: topic=%s partition=%d offset=%d error=%v",
                    msg.Topic, msg.Partition, msg.Offset, err)
                // Мусорное сообщение — коммитим, чтобы не блокировать очередь
                if err := c.reader.CommitMessages(ctx, msg); err != nil {
                    log.Printf("commit error: %v", err)
                }
                continue
            }

            // 2. Валидация заказа
            if err := validateOrder(order); err != nil {
                log.Printf("invalid order: %v, topic=%s partition=%d offset=%d",
                    err, msg.Topic, msg.Partition, msg.Offset)
                // Невалидные данные — коммитим (позже можно в DLQ)
                if err := c.reader.CommitMessages(ctx, msg); err != nil {
                    log.Printf("commit error: %v", err)
                }
                continue
            }

            // 3. Сохранение с ретраями
            if err := c.saveWithRetry(ctx, order); err != nil {
                log.Printf("failed to save order after %d attempts: %v, stopping consumer", maxRetries, err)
                // Не коммитим offset. Завершаем процесс, чтобы при рестарте сообщение было перечитано.
                // В реальном приложении лучше отправить в DLQ и продолжить, но для гарантии
                // отсутствия потери данных в учебном проекте останавливаем consumer.
                panic(fmt.Sprintf("unable to process message offset %d: %v", msg.Offset, err))
            }

            // 4. Обновляем кэш
            c.cache.Set(order.OrderUID, &order)

            // 5. Коммитим успешно обработанное сообщение
            if err := c.reader.CommitMessages(ctx, msg); err != nil {
                log.Printf("commit error: %v", err)
            }
        }
    }()
}

func (c *Consumer) saveWithRetry(ctx context.Context, order model.Order) error {
    var err error
    delay := baseDelay
    for attempt := 0; attempt < maxRetries; attempt++ {
        err = c.store.SaveOrder(ctx, order)
        if err == nil {
            return nil
        }
        log.Printf("save attempt %d failed: %v", attempt+1, err)
        if ctx.Err() != nil {
            return ctx.Err()
        }
        time.Sleep(delay)
        delay *= 2
    }
    return errors.New("exhausted retries")
}

func (c *Consumer) Close() error {
    return c.reader.Close()
}
