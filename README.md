# EFFM – сервис учёта онлайн-подписок

REST-сервис для агрегации и управления онлайн-подписками пользователей.  
Поддерживает CRUDL-операции и подсчёт суммарной стоимости подписок за период.  
Стек: Go, Gin, PostgreSQL, pgx, Swagger, Viper, Docker Compose.

---

## Возможности

- Создание записи о подписке за конкретный месяц  
- Получение одной подписки или списка (с фильтром по user_id)  
- Частичное обновление (service_name, price)  
- Удаление подписки  
- Подсчёт общей стоимости за период (YYYY-MM…YYYY-MM)  
- Миграции PostgreSQL  
- Конфигурация через config.yaml и .env  
- Swagger-документация  
- Запуск через Docker Compose

---

## Запуск проекта

### 1. Клонирование репозитория

```bash
git clone https://github.com/maaw77/effm.git
cd effm
```

### 2. Создать файл `.env`

Перед запуском Docker необходимо создать файл `.env`:

```
POSTGRES_DB=postgres
POSTGRES_USER=postgres
POSTGRES_PASSWORD=epas

# Для работы в Docker
HOST_DB=db
PORT_DB=5432
```

### 3. Запуск окружения

```bash
docker compose up --build
```

После запуска сервис будет доступен на:

```
http://localhost:8080
```

### 4. Swagger UI

```
http://localhost:8080/swagger/index.html
```

---

## Основные эндпоинты

### POST /api/subscriptions

Создание записи о подписке.

Пример:

```json
{
  "service_name": "Yandex Plus",
  "price": 400,
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
  "year_month": "2025-07"
}
```

### GET /api/subscriptions  
Список всех подписок, опционально с фильтром по user_id.

### GET /api/subscriptions/{id}  
Получение одной подписки.

### PUT /api/subscriptions/{id}  
Частичное обновление записи.

### DELETE /api/subscriptions/{id}  
Удаление подписки.

### GET /api/subscriptions/total  
Подсчёт общей стоимости за период.

Параметры:

```
start=YYYY-MM
end=YYYY-MM
user_id (optional)
service (optional)
```

---

## Структура проекта

```
effm/
├── cmd/server/
├── config/
├── internal/
│   ├── database/
│   ├── dto/
│   ├── models/
│   └── server/
├── migrations/
└── docker-compose.yml
```

---

## Тестирование

Запуск модульных тестов:

```bash
go test ./...
```

---

## Быстрая проверка API (Linux)

Для тестирования API после запуска контейнеров можно использовать скрипт:

```bash
./test-api.sh
```

---

## Конфигурация

- `config/config.yaml` — основные настройки сервера и базы данных  
- `.env` — параметры PostgreSQL для Docker  

---

## Миграции

```bash
go run -tags migrate cmd/server/migrate.go
```

---

## Лицензия

MIT
