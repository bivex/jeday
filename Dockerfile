FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build statically linked binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth-worker ./cmd/worker

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/auth-api .
COPY --from=builder /app/auth-worker .
COPY .env .env

EXPOSE 8080

CMD ["./auth-api"]
