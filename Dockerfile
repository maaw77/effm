# ---------- СТАДИЯ СБОРКИ ----------
FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

# Копируем весь проект
COPY . /build

WORKDIR /build

# Загружаем зависимости
RUN go mod tidy

# Каталог для итогового бинарника
RUN mkdir /app

# Собираем бинарник с тегом migrate
RUN go build -tags migrate -o /app/effm ./cmd/server/

# ---------- СТАДИЯ РАНТАЙМА ----------
FROM alpine:latest

RUN apk add --no-cache ca-certificates

# Создаем рабочие каталоги
RUN mkdir -p /app/config
RUN mkdir -p /app/docs
RUN mkdir -p /app/migrations

WORKDIR /app

# Бинарник
COPY --from=builder /app/effm /app/effm

# Конфиг
COPY ./config/config.yaml ./config/

# Swagger-файлы
COPY ./docs/swagger.yaml ./docs/
COPY ./docs/docs.go ./docs/  

# Миграции
COPY ./migrations/* ./migrations/

# .env, если он используется
COPY .env .

# Запуск
CMD ["/app/effm"]
