package main

import (
  "github.com/nats-io/stan.go"
  "log"
)

func main() {
  data := `{
    "order_uid": "b563feb7b2b84b6test",
    "track_number": "WBILMTESTTRACK",
    "entry": "WBIL",
    "delivery": {
      "name": "Иван Иванов",
      "phone": "+7 (999) 123-45-67",
      "zip": "125009",
      "city": "Москва",
      "address": "ул. Тверская, д. 15",
      "region": "Москва",
      "email": "ivan@example.com"
    },
    "payment": {
      "transaction": "b563feb7b2b84b6test",
      "request_id": "",
      "currency": "RUB",
      "provider": "wbpay",
      "amount": 1817,
      "payment_dt": 1637907727,
      "bank": "альфа-банк",
      "delivery_cost": 1500,
      "goods_total": 317,
      "custom_fee": 0
    },
    "items": [
      {
        "chrt_id": 9934930,
        "track_number": "WBILMTESTTRACK",
        "price": 453,
        "rid": "ab4219087a764ae0btest",
        "name": "Тушь для ресниц",
        "sale": 30,
        "size": "0",
        "total_price": 317,
        "nm_id": 2389212,
        "brand": "Vivienne Sabo",
        "status": 202
      }
    ],
    "locale": "ru",
    "internal_signature": "",
    "customer_id": "test",
    "delivery_service": "meest",
    "shardkey": "9",
    "sm_id": 99,
    "date_created": "2021-11-26T06:22:19Z",
    "oof_shard": "1"
  }`

  sc, err := stan.Connect("test-cluster", "publisher", stan.NatsURL("nats://localhost:4223"))
  if err != nil {
    log.Fatal(err)
  }
  defer sc.Close()

  err = sc.Publish("orders", []byte(data))
  if err != nil {
    log.Fatal(err)
  }
  log.Println(" Тестовый заказ отправлен в NATS!")
}
