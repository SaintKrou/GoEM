# Stage 1: сборка
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем модули и зависимостей для кэширования
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o subscription-service ./cmd

# Stage 2: финальный образ
FROM alpine:latest

# Установим ca-certificates для HTTPS и tzdata для временных зон
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Копируем бинарник и статику (Swagger)
COPY --from=builder /app/subscription-service .
COPY --from=builder /app/docs ./docs

# Порт приложения
EXPOSE 8080

# Запуск
CMD ["./subscription-service"]