FROM golang:alpine AS builder

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Сборка с оптимизацией размера и удалением отладочной информации (-ldflags="-s -w")
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo -o auth-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo -o auth-worker ./cmd/worker

FROM alpine:latest

WORKDIR /app

# Установка часовых поясов и сертификатов + создание не-root пользователя
RUN apk --no-cache add ca-certificates tzdata \
    && adduser -D -u 1000 jedi

COPY --from=builder /app/auth-api .
COPY --from=builder /app/auth-worker .
COPY .env .env

# Даем права пользователю
RUN chown -R jedi:jedi /app

# Переключаемся на пользователя
USER jedi

# GOMEMLIMIT предотвращает OOM, оставляя запас для ОС
# GOGC настраивает частоту сборки мусора для высоконагруженных систем
ENV GOMEMLIMIT=512MiB
ENV GOGC=100

EXPOSE 8080

CMD ["./auth-api"]
