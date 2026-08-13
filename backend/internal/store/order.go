package store

import (
    "context"
    "order-service/internal/model"
)

func (s *Store) SaveOrder(ctx context.Context, order model.Order) error {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx) // откат, если не закоммитим

    // Вставка в таблицу orders
    _, err = tx.Exec(ctx, `
        INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, 
            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (order_uid) DO UPDATE SET
            track_number = EXCLUDED.track_number,
            entry = EXCLUDED.entry,
            locale = EXCLUDED.locale,
            internal_signature = EXCLUDED.internal_signature,
            customer_id = EXCLUDED.customer_id,
            delivery_service = EXCLUDED.delivery_service,
            shardkey = EXCLUDED.shardkey,
            sm_id = EXCLUDED.sm_id,
            date_created = EXCLUDED.date_created,
            oof_shard = EXCLUDED.oof_shard
    `,
        order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
        order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard,
    )
    if err != nil {
        return err
    }

    // Вставка delivery
    _, err = tx.Exec(ctx, `
        INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (order_uid) DO UPDATE SET
            name = EXCLUDED.name,
            phone = EXCLUDED.phone,
            zip = EXCLUDED.zip,
            city = EXCLUDED.city,
            address = EXCLUDED.address,
            region = EXCLUDED.region,
            email = EXCLUDED.email
    `,
        order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
        order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
    )
    if err != nil {
        return err
    }

    // Вставка payment
    _, err = tx.Exec(ctx, `
        INSERT INTO payment (order_uid, transaction, request_id, currency, provider, 
            amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (order_uid) DO UPDATE SET
            transaction = EXCLUDED.transaction,
            request_id = EXCLUDED.request_id,
            currency = EXCLUDED.currency,
            provider = EXCLUDED.provider,
            amount = EXCLUDED.amount,
            payment_dt = EXCLUDED.payment_dt,
            bank = EXCLUDED.bank,
            delivery_cost = EXCLUDED.delivery_cost,
            goods_total = EXCLUDED.goods_total,
            custom_fee = EXCLUDED.custom_fee
    `,
        order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
        order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank,
        order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee,
    )
    if err != nil {
        return err
    }

    // Удаляем старые items и вставляем новые (чтобы синхронизировать)
    _, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid = $1`, order.OrderUID)
    if err != nil {
        return err
    }
    for _, item := range order.Items {
        _, err = tx.Exec(ctx, `
            INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, 
                size, total_price, nm_id, brand, status)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        `,
            order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.RID, item.Name,
            item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
        )
        if err != nil {
            return err
        }
    }

    return tx.Commit(ctx)
}

func (s *Store) GetOrderByUID(ctx context.Context, uid string) (*model.Order, error) {
    order := &model.Order{}
    // Основная информация
    err := s.pool.QueryRow(ctx, `
        SELECT order_uid, track_number, entry, locale, internal_signature, customer_id,
               delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders WHERE order_uid = $1
    `, uid).Scan(
        &order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
        &order.InternalSignature, &order.CustomerID, &order.DeliveryService,
        &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard,
    )
    if err != nil {
        return nil, err
    }

    // Delivery
    err = s.pool.QueryRow(ctx, `
        SELECT name, phone, zip, city, address, region, email
        FROM delivery WHERE order_uid = $1
    `, uid).Scan(
        &order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
        &order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region,
        &order.Delivery.Email,
    )
    if err != nil {
        return nil, err
    }

    // Payment
    err = s.pool.QueryRow(ctx, `
        SELECT transaction, request_id, currency, provider, amount, payment_dt,
               bank, delivery_cost, goods_total, custom_fee
        FROM payment WHERE order_uid = $1
    `, uid).Scan(
        &order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
        &order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
        &order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal,
        &order.Payment.CustomFee,
    )
    if err != nil {
        return nil, err
    }

    // Items
    rows, err := s.pool.Query(ctx, `
        SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
        FROM items WHERE order_uid = $1
    `, uid)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    for rows.Next() {
        var item model.Item
        err := rows.Scan(
            &item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name,
            &item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
        )
        if err != nil {
            return nil, err
        }
        order.Items = append(order.Items, item)
    }
    return order, nil
}

func (s *Store) GetAllOrders(ctx context.Context) ([]*model.Order, error) {
    // Здесь можно получить список order_uid, а затем по каждому вызвать GetOrderByUID.
    // Для простоты сделаем так.
    rows, err := s.pool.Query(ctx, `SELECT order_uid FROM orders`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var orders []*model.Order
    for rows.Next() {
        var uid string
        if err := rows.Scan(&uid); err != nil {
            return nil, err
        }
        order, err := s.GetOrderByUID(ctx, uid)
        if err != nil {
            continue // или возвращать ошибку, но лучше пропустить битый заказ
        }
        orders = append(orders, order)
    }
    return orders, nil
}