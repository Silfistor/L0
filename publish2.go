package main

import (
  "github.com/nats-io/stan.go"
  "log"
)

func main() {
  data := `{
    "order_uid": "e123fgh4i5j6k7l8test",
    "track_number": "WBILMTRACK002",
    "entry": "WBIL",
    "delivery": {
      "name": "Михаил Сидоров",
      "phone": "+7 (916) 777-88-99",
      "zip": "450000",
      "city": "Уфа",
      "address": "пр. Октября, д. 100, подъезд 3",
      "region": "Республика Башкортостан",
      "email": "m.sidorov@mail.ru"
    },
    "payment": {
      "transaction": "e123fgh4i5j6k7l8test",
      "request_id": "",
      "currency": "RUB",
      "provider": "wbpay",
      "amount": 5420,
      "payment_dt": 1713456789,
      "bank": "т-банк",
      "delivery_cost": 0,
      "goods_total": 5420,
      "custom_fee": 0
    },
    "items": [
      {
        "chrt_id": 9956789,
        "track_number": "WBILMTRACK002",
        "price": 2990,
        "rid": "ef6431209c986cg2dtest",
        "name": "Конструктор LEGO Technic",
        "sale": 0,
        "size": "—",
        "total_price": 2990,
        "nm_id": 9876543,
        "brand": "LEGO",
        "status": 202
      },
      {
        "chrt_id": 9956790,
        "track_number": "WBILMTRACK002",
        "price": 1250,
        "rid": "ef6431209c986cg2dtest",
        "name": "Мягкая игрушка — единорог",
        "sale": 20,
        "size": "40 см",
        "total_price": 1000,
        "nm_id": 9876544,
        "brand": "Tiger Family",
        "status": 202
      },
      {
        "chrt_id": 9956791,
        "track_number": "WBILMTRACK002",
        "price": 1800,
        "rid": "ef6431209c986cg2dtest",
        "name": "Настольная игра «Монополия»",
        "sale": 21,
        "size": "—",
        "total_price": 1430,
        "nm_id": 9876545,
        "brand": "Hasbro",
        "status": 202
      }
    ],
    "locale": "ru",
    "internal_signature": "",
    "customer_id": "customer_456",
    "delivery_service": "boxberry",
    "shardkey": "3",
    "sm_id": 77,
    "date_created": "2024-04-18T09:33:09Z",
    "oof_shard": "1"
  }`

  sc, err := stan.Connect("test-cluster", "publisher3", stan.NatsURL("nats://localhost:4223"))
  if err != nil {
    log.Fatal(err)
  }
  defer sc.Close()

  err = sc.Publish("orders", []byte(data))
  if err != nil {
    log.Fatal(err)
  }
  log.Println(" Заказ детских товаров (ID: e123fgh4i5j6k7l8test) отправлен в NATS!")
}
