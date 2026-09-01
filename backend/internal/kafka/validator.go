package kafka

import (
    "errors"
    "fmt"
    "order-service/internal/model"
)

func validateOrder(order model.Order) error {
    if order.OrderUID == "" {
        return errors.New("order_uid is empty")
    }
    if order.Delivery == (model.Delivery{}) {
        return errors.New("delivery is missing")
    }
    if order.Payment == (model.Payment{}) {
        return errors.New("payment is missing")
    }
    if len(order.Items) == 0 {
        return errors.New("items are empty")
    }
    if order.Payment.Transaction == "" {
        return errors.New("payment.transaction is empty")
    }
    if order.Payment.Amount < 0 || order.Payment.DeliveryCost < 0 ||
        order.Payment.GoodsTotal < 0 || order.Payment.CustomFee < 0 {
        return errors.New("negative monetary value")
    }
    // Проверка согласованности суммы товаров
    var sumTotalPrice float64
    for _, item := range order.Items {
        sumTotalPrice += item.TotalPrice
    }
    if sumTotalPrice > 0 && order.Payment.GoodsTotal != int(sumTotalPrice) {
        return fmt.Errorf("goods_total (%d) does not match sum of items total_price (%f)",
            order.Payment.GoodsTotal, sumTotalPrice)
    }
    // Проверка обязательных полей в delivery
    if order.Delivery.Name == "" || order.Delivery.Phone == "" || order.Delivery.Address == "" {
        return errors.New("delivery fields name/phone/address are required")
    }
    // Проверка items
    for i, item := range order.Items {
        if item.Name == "" || item.RID == "" || item.TrackNumber == "" {
            return fmt.Errorf("item %d: name/rid/track_number are required", i)
        }
        if item.Price < 0 || item.TotalPrice < 0 {
            return fmt.Errorf("item %d: negative price", i)
        }
    }
    return nil
}
