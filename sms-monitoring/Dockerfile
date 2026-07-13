FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /monitoring-worker ./cmd/monitoring-worker

FROM alpine:latest
WORKDIR /app
COPY --from=builder /monitoring-worker /app/monitoring-worker
CMD ["/app/monitoring-worker"]
