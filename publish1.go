package main

import (
  "github.com/nats-io/stan.go"
  "log"
)

func main() {
  data := `{
    "order_uid": "c789def0a1b2c3d4test",
    "track_number": "WBILMTRACK001",
    "entry": "WBIL",
    "delivery": {
      "name": "Анна Петрова",
      "phone": "+7 (905) 555-12-34",
      "zip": "630099",
      "city": "Новосибирск",
      "address": "ул. Ленина, д. 42, кв. 15",
      "region": "Новосибирская область",
      "email": "anna.p@example.com"
    },
    "payment": {
      "transaction": "c789def0a1b2c3d4test",
      "request_id": "",
      "currency": "RUB",
      "provider": "wbpay",
      "amount": 3250,
      "payment_dt": 1712345678,
      "bank": "сбербанк",
      "delivery_cost": 300,
      "goods_total": 2950,
      "custom_fee": 0
    },
    "items": [
      {
        "chrt_id": 8845671,
        "track_number": "WBILMTRACK001",
        "price": 1500,
        "rid": "cd5320198b875bf1ctest",
        "name": "Джинсы мужские",
        "sale": 10,
        "size": "L",
        "total_price": 1350,
        "nm_id": 1234567,
        "brand": "Levi's",
        "status": 202
      },
      {
        "chrt_id": 8845672,
        "track_number": "WBILMTRACK001",
        "price": 1800,
        "rid": "cd5320198b875bf1ctest",
        "name": "Футболка хлопковая",
        "sale": 15,
        "size": "M",
        "total_price": 1600,
        "nm_id": 1234568,
        "brand": "Zara",
        "status": 202
      }
    ],
    "locale": "ru",
    "internal_signature": "",
    "customer_id": "customer_789",
    "delivery_service": "cdek",
    "shardkey": "5",
    "sm_id": 88,
    "date_created": "2024-04-05T14:27:58Z",
    "oof_shard": "2"
  }`

  sc, err := stan.Connect("test-cluster", "publisher2", stan.NatsURL("nats://localhost:4223"))
  if err != nil {
    log.Fatal(err)
  }
  defer sc.Close()

  err = sc.Publish("orders", []byte(data))
  if err != nil {
    log.Fatal(err)
  }
  log.Println("Заказ одежды (ID: c789def0a1b2c3d4test) отправлен в NATS!")
}
