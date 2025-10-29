package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/nats-io/stan.go"
)

// === Модели данных ===
type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type Payment struct {
	Transaction   string `json:"transaction"`
	RequestID     string `json:"request_id"`
	Currency      string `json:"currency"`
	Provider      string `json:"provider"`
	Amount        int    `json:"amount"`
	PaymentDt     int64  `json:"payment_dt"`
	Bank          string `json:"bank"`
	DeliveryCost  int    `json:"delivery_cost"`
	GoodsTotal    int    `json:"goods_total"`
	CustomFee     int    `json:"custom_fee"`
}

type Item struct {
	ChrtID      int64  `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int64  `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

type Order struct {
	OrderUID          string    `json:"order_uid"`
	TrackNumber       string    `json:"track_number"`
	Entry             string    `json:"entry"`
	Delivery          Delivery  `json:"delivery"`
	Payment           Payment   `json:"payment"`
	Items             []Item    `json:"items"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	Shardkey          string    `json:"shardkey"`
	SmID              int       `json:"sm_id"`
	DateCreated       time.Time `json:"date_created"`
	OofShard          string    `json:"oof_shard"`
}

// === Глобальные переменные ===
var db *sql.DB
var orderCache = make(map[string]Order)
var cacheMutex sync.RWMutex

// === Инициализация БД (порт 5433) ===
func initDB() {
	connStr := "user=orderuser password=orderpass dbname=orderdb sslmode=disable host=localhost port=5433"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(" Не удалось подключиться к БД:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(" Не удалось пингануть БД:", err)
	}
	log.Println(" Подключение к PostgreSQL установлено")
}

// === Сохранение заказа в БД ===
func saveOrderToDB(order Order) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO payments (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return err
	}

	for _, item := range order.Items {
		_, err = tx.Exec(`
			INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// === Загрузка одного заказа из БД ===
func getOrderFromDB(uid string) (Order, error) {
	var order Order
	order.OrderUID = uid

	err := db.QueryRow(`
		SELECT track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid = $1`, uid).
		Scan(&order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
			&order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
	if err != nil {
		return order, err
	}

	err = db.QueryRow(`
		SELECT name, phone, zip, city, address, region, email
		FROM deliveries WHERE order_uid = $1`, uid).
		Scan(&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
			&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email)
	if err != nil {
		return order, err
	}

	err = db.QueryRow(`
		SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
		FROM payments WHERE order_uid = $1`, uid).
		Scan(&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
			&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt, &order.Payment.Bank,
			&order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee)
	if err != nil {
		return order, err
	}

	rows, err := db.Query(`
		SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items WHERE order_uid = $1`, uid)
	if err != nil {
		return order, err
	}
	defer rows.Close()

	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status)
		if err != nil {
			return order, err
		}
		order.Items = append(order.Items, item)
	}

	return order, nil
}

// === Восстановление кэша из БД ===
func loadCacheFromDB() {
	rows, err := db.Query("SELECT order_uid FROM orders")
	if err != nil {
		log.Println(" Ошибка при загрузке UID из БД:", err)
		return
	}
	defer rows.Close()

	var uids []string
	for rows.Next() {
		var uid string
		rows.Scan(&uid)
		uids = append(uids, uid)
	}

	for _, uid := range uids {
		order, err := getOrderFromDB(uid)
		if err != nil {
			log.Printf(" Не удалось загрузить заказ %s из БД: %v", uid, err)
			continue
		}
		cacheMutex.Lock()
		orderCache[uid] = order
		cacheMutex.Unlock()
	}
	log.Printf(" Кэш восстановлен из БД: %d заказов", len(uids))
}

// === Подписка на NATS Streaming (порт 4223) ===
func startNATSSubscriber() {
	sc, err := stan.Connect("test-cluster", "order-service", stan.NatsURL("nats://localhost:4223"))
	if err != nil {
		log.Fatal(" NATS Streaming connect error:", err)
	}
	defer sc.Close()

	_, err = sc.Subscribe("orders", func(msg *stan.Msg) {
		var order Order
		if err := json.Unmarshal(msg.Data, &order); err != nil {
			log.Printf(" Невалидный JSON: %v", err)
			return
		}

		if order.OrderUID == "" {
                        log.Println(" Отклонено: отсутствует order_uid")
                        return
                }
                if len(order.OrderUID) > 100 {
                        log.Println(" Отклонено: order_uid слишком длинный")
                        return
                }
                if order.Delivery.Name == "" {
                        log.Println(" Отклонено: отсутствует имя получателя")
                        return
                }
                if len(order.Items) == 0 {
                        log.Println(" Отклонено: заказ без товаров")
                        return
                }
                if order.Payment.Amount <= 0 {
                        log.Println(" Отклонено: некорректная сумма оплаты")
                        return
                }
		if err := saveOrderToDB(order); err != nil {
			log.Printf(" Ошибка сохранения в БД: %v", err)
			return
		}

		cacheMutex.Lock()
		orderCache[order.OrderUID] = order
		cacheMutex.Unlock()

		log.Printf(" Заказ %s сохранён и закэширован", order.OrderUID)
	}, stan.DurableName("order-durable"))

	if err != nil {
		log.Fatal(" Ошибка подписки на NATS:", err)
	}

	select {}
}

// === HTTP-обработчики ===
func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "order_uid")

	cacheMutex.RLock()
	order, exists := orderCache[uid]
	cacheMutex.RUnlock()

	if !exists {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
func clearAllHandler(w http.ResponseWriter, r *http.Request) {
  // Очистка БД
  _, err := db.Exec(`
    DELETE FROM items;
    DELETE FROM payments;
    DELETE FROM deliveries;
    DELETE FROM orders;
  `)
  if err != nil {
    http.Error(w, "Ошибка очистки БД: "+err.Error(), http.StatusInternalServerError)
    return
  }

  // Очистка кэша
  cacheMutex.Lock()
  orderCache = make(map[string]Order) 
  cacheMutex.Unlock()

  w.WriteHeader(http.StatusOK)
  w.Write([]byte(" Все данные удалены из БД и кэша.\n"))
}
func getUIHandler(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "order_uid")
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Заказ %s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f8f9fa;
            margin: 0;
            padding: 20px;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.08);
            overflow: hidden;
        }
        header {
            background: #4361ee;
            color: white;
            padding: 20px 24px;
        }
        h1 {
            margin: 0;
            font-size: 1.5rem;
            font-weight: 600;
        }
        .content {
            padding: 24px;
        }
        .section {
            margin-bottom: 24px;
        }
        .section h2 {
            font-size: 1.2rem;
            color: #333;
            margin-bottom: 12px;
            padding-bottom: 6px;
            border-bottom: 1px solid #eee;
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 12px;
        }
        .field {
            display: flex;
            flex-direction: column;
        }
        .field-label {
            font-size: 0.85rem;
            color: #666;
            margin-bottom: 4px;
        }
        .field-value {
            font-weight: 500;
            color: #222;
        }
        .items-list {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }
        .item-card {
            background: #f8f9fa;
            padding: 12px;
            border-radius: 8px;
            border-left: 4px solid #4cc9f0;
        }
        pre {
            background: #2b2b2b;
            color: #f8f8f2;
            padding: 16px;
            border-radius: 8px;
            overflow-x: auto;
            font-size: 13px;
        }
        .error {
            color: #e63946;
            padding: 20px;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Детали заказа: %s</h1>
        </header>
        <div class="content">
            <div id="order-data">Загрузка...</div>
        </div>
    </div>

    <script>
        fetch('/order/%s')
            .then(res => {
                if (!res.ok) throw new Error('Заказ не найден');
                return res.json();
            })
            .then(order => {
                document.getElementById('order-data').innerHTML = renderOrder(order);
            })
            .catch(err => {
                document.getElementById('order-data').innerHTML = 
                    '<div class="error"><h3>❌ ' + err.message + '</h3><p>Проверьте правильность ID заказа.</p></div>';
            });

        function renderOrder(o) {
            let html = '';

            // Общая информация
            html += '<div class="section"><h2>Общее</h2><div class="grid">';
            html += field('ID заказа', o.order_uid);
            html += field('Трек-номер', o.track_number);
            html += field('Точка входа', o.entry);
            html += field('Язык', o.locale);
            html += field('ID клиента', o.customer_id);
            html += field('Служба доставки', o.delivery_service);
            html += field('Создан', new Date(o.date_created).toLocaleString('ru-RU'));
            html += '</div></div>';

            // Доставка
            html += '<div class="section"><h2>Доставка</h2><div class="grid">';
            html += field('Имя', o.delivery.name);
            html += field('Телефон', o.delivery.phone);
            html += field('Email', o.delivery.email); // ← остаётся на английском
            html += field('Адрес', o.delivery.address + ', ' + o.delivery.city + ', ' + o.delivery.region + ', ' + o.delivery.zip);
            html += '</div></div>';

            // Оплата
            html += '<div class="section"><h2>Оплата</h2><div class="grid">';
            html += field('Сумма', o.payment.amount + ' ' + o.payment.currency);
            html += field('Провайдер', o.payment.provider);
            html += field('Банк', o.payment.bank);
            html += field('Стоимость доставки', o.payment.delivery_cost);
            html += field('Стоимость товаров', o.payment.goods_total);
            html += field('Оплачено', new Date(o.payment.payment_dt * 1000).toLocaleString('ru-RU'));
            html += '</div></div>';

            // Товары
            html += '<div class="section"><h2>Товары (' + o.items.length + ')</h2><div class="items-list">';
            o.items.forEach(item => {
                html += '<div class="item-card">';
                html += '<strong>' + item.name + '</strong> (' + item.brand + ')<br>';
                html += 'Цена: ' + item.price + ' → Итого: ' + item.total_price + ' (' + item.sale + '%% скидка)<br>';
                html += 'Размер: ' + item.size + ' | Статус: ' + item.status;
                html += '</div>';
            });
            html += '</div></div>';

            // Сырой JSON
            html += '<details><summary>🔍 Показать исходный JSON</summary><pre>' + JSON.stringify(o, null, 2) + '</pre></details>';

            return html;
        }

        function field(label, value) {
            return '<div class="field"><div class="field-label">' + label + '</div><div class="field-value">' + (value || '—') + '</div></div>';
        }
    </script>
</body>
</html>`, uid, uid, uid)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем все ID из кэша
	cacheMutex.RLock()
	uids := make([]string, 0, len(orderCache))
	for uid := range orderCache {
		uids = append(uids, uid)
	}
	cacheMutex.RUnlock()

	// Если кэш пуст — попробуем загрузить из БД (на случай, если сервис только запустился)
	if len(uids) == 0 {
		rows, err := db.Query("SELECT order_uid FROM orders ORDER BY date_created DESC")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var uid string
				rows.Scan(&uid)
				uids = append(uids, uid)
			}
		}
	}

	// Формируем HTML
	html := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Все заказы</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f8f9fa;
            margin: 0;
            padding: 20px;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.08);
            overflow: hidden;
        }
        header {
            background: #4361ee;
            color: white;
            padding: 20px;
            text-align: center;
        }
        h1 {
            margin: 0;
            font-size: 1.6rem;
        }
        .content {
            padding: 20px;
        }
        .manual-search {
            margin-bottom: 24px;
            padding: 16px;
            background: #f1f3f5;
            border-radius: 8px;
        }
        .manual-search input {
            width: 100%;
            padding: 10px;
            font-size: 16px;
            border: 1px solid #ccc;
            border-radius: 6px;
            box-sizing: border-box;
        }
        .manual-search button {
            margin-top: 8px;
            width: 100%;
            padding: 10px;
            background: #4cc9f0;
            color: white;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 16px;
        }
        .orders-list {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }
        .order-item {
            padding: 12px;
            background: #f8f9fa;
            border-radius: 8px;
            border-left: 4px solid #4361ee;
        }
        .order-item a {
            text-decoration: none;
            color: #4361ee;
            font-weight: 600;
            font-family: monospace;
        }
        .order-item a:hover {
            text-decoration: underline;
        }
        .empty {
            text-align: center;
            color: #666;
            padding: 40px 0;
        }
        .refresh {
            text-align: center;
            margin-top: 20px;
        }
        .refresh button {
            background: #7209b7;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 6px;
            cursor: pointer;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>📦 Все заказы</h1>
        </header>
        <div class="content">
            <div class="manual-search">
                <input type="text" id="manualId" placeholder="Или введите ID заказа вручную...">
                <button onclick="goToOrder()">Перейти</button>
            </div>

            <h2>Список заказов (всего: ` + fmt.Sprintf("%d", len(uids)) + `)</h2>
`

	if len(uids) == 0 {
		html += `<div class="empty">Нет заказов. Отправьте данные через NATS.</div>`
	} else {
		html += `<div class="orders-list">`
		for _, uid := range uids {
			html += fmt.Sprintf(`<div class="order-item"><a href="/ui/%s">%s</a></div>`, uid, uid)
		}
		html += `</div>`
	}

	html += `
            <div class="refresh">
                <button onclick="location.reload()">🔄 Обновить список</button>
            </div>
        </div>
    </div>

    <script>
        function goToOrder() {
            const id = document.getElementById('manualId').value.trim();
            if (id) {
                window.location.href = '/ui/' + encodeURIComponent(id);
            }
        }
        document.getElementById('manualId').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') goToOrder();
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
// === MAIN ===
func main() {
  
  log.Println(" Инициализация базы данных...")
  initDB()

  _, err := db.Exec(`
    CREATE TABLE IF NOT EXISTS orders (
      order_uid TEXT PRIMARY KEY,
      track_number TEXT,
      entry TEXT,
      locale TEXT,
      internal_signature TEXT,
      customer_id TEXT,
      delivery_service TEXT,
      shardkey TEXT,
      sm_id INTEGER,
      date_created TIMESTAMPTZ,
      oof_shard TEXT
    );
    CREATE TABLE IF NOT EXISTS deliveries (
      order_uid TEXT REFERENCES orders(order_uid) ON DELETE CASCADE,
      name TEXT,
      phone TEXT,
      zip TEXT,
      city TEXT,
      address TEXT,
      region TEXT,
      email TEXT
    );
    CREATE TABLE IF NOT EXISTS payments (
      order_uid TEXT REFERENCES orders(order_uid) ON DELETE CASCADE,
      transaction TEXT,
      request_id TEXT,
      currency TEXT,
      provider TEXT,
      amount INTEGER,
      payment_dt BIGINT,
      bank TEXT,
      delivery_cost INTEGER,
      goods_total INTEGER,
      custom_fee INTEGER
    );
    CREATE TABLE IF NOT EXISTS items (
      order_uid TEXT REFERENCES orders(order_uid) ON DELETE CASCADE,
      chrt_id BIGINT,
      track_number TEXT,
      price INTEGER,
      rid TEXT,
      name TEXT,
      sale INTEGER,
      size TEXT,
      total_price INTEGER,
      nm_id BIGINT,
      brand TEXT,
      status INTEGER
    );
  `)
  if err != nil {
    log.Fatal(" Ошибка создания таблиц:", err)
  }

  loadCacheFromDB()

  log.Println(" Подключение к NATS Streaming...")
  go startNATSSubscriber()

  r := chi.NewRouter()
  r.Get("/", homeHandler)
  r.Get("/order/{order_uid}", getOrderHandler)
  r.Get("/ui/{order_uid}", getUIHandler)

  log.Println(" HTTP-сервер запущен на http://localhost:8080")
  log.Fatal(http.ListenAndServe(":8080", r))
}
