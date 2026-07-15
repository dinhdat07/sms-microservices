FROM golang:1.26.2-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /monitoring-worker ./cmd/monitoring-worker

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --chown=nonroot:nonroot --from=builder /monitoring-worker /app/monitoring-worker

USER nonroot:nonroot

CMD ["/app/monitoring-worker"]
