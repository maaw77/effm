# ========== BUILDER ==========
FROM golang:1.25-alpine AS builder

# Устанавливаем только то, что нужно для сборки
RUN apk add --no-cache git

WORKDIR /build

# Кэшируем зависимости (это главное ускорение при пересборке)
COPY go.mod go.sum ./
RUN go mod download

# Копируем остальной код
COPY . .

# Собираем статический бинарник с миграциями
RUN CGO_ENABLED=0 GOOS=linux go build -tags migrate -ldflags="-s -w" -o /app/effm ./cmd/server/

# ========== RUNTIME ==========
FROM alpine:3.20 AS runtime

# ca-certificates — для HTTPS, tzdata — чтобы время было московское (а не UTC-3)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Копируем только то, что реально нужно во время выполнения
COPY --from=builder /app/effm /app/effm
COPY config/config.yaml ./config/
COPY docs/ ./docs/
COPY migrations/ ./migrations/


# Явно указываем таймзону
ENV TZ=Europe/Moscow

EXPOSE 8080

CMD ["/app/effm"]