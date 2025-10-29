# Сервис заказов
Простой сервис на Go, который:
- Подписывается на канал NATS Streaming и принимает заказы в формате JSON
- Сохраняет данные в PostgreSQL
- Кэширует заказы в памяти
- Восстанавливает кэш из БД при перезапуске
- Предоставляет веб-интерфейс для просмотра заказов

### Требования
- [Docker](https://www.docker.com/products/docker-desktop)
- [Go 1.21+](https://go.dev/dl/)

### 1. Запустите зависимости
```bash
# PostgreSQL (порт 5433 на хосте)
docker run --name order-db -e POSTGRES_USER=orderuser -e POSTGRES_PASSWORD=orderpass -e POSTGRES_DB=orderdb -p 5433:5432 -d postgres:15

# NATS Streaming (порт 4223 на хосте)
docker run -d --name nats-streaming -p 4223:4222 -p 8223:8222 nats-streaming:0.25.6 -m 8222 -store file -dir /data/store -cluster_id test-cluster
### 2. Создание и запуск сервера, После чего необходимо будет перетащить внутрь файлы
mkdir order-service-demo
cd order-service-demo
go mod init order-service-demo
### Запуск файлов
go run main.go

### 3. Откройте в браузере
Список заказов: http://localhost:8080

### 4. Отправьте тестовый заказ
go run publish.go      # базовый заказ
go run publish2.go     # заказ одежды
go run publish3.go     # заказ детских товаров

###  Очистка данных
docker exec -it order-db psql -U orderuser -d orderdb -c "
    DELETE FROM items;
    DELETE FROM payments;
    DELETE FROM deliveries;
    DELETE FROM orders;
"
