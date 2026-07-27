FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o agent-handler cmd/agent-handler/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/agent-handler .

EXPOSE 8084

CMD ["./agent-handler"]
