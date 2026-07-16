FROM golang:1.26.2-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /monitoring-worker ./cmd/monitoring-worker

FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache iproute2

COPY --from=builder /monitoring-worker /app/monitoring-worker
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

CMD ["/app/entrypoint.sh"]
