FROM golang:1.26.2-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server-simulator ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /server-seed ./cmd/seed

FROM alpine:3.19

RUN apk add --no-cache nftables iproute2

WORKDIR /app
COPY entrypoint.sh .
COPY --from=builder /server-simulator /app/server-simulator
COPY --from=builder /server-seed /app/server-seed

RUN sed -i 's/\r$//' entrypoint.sh && chmod +x entrypoint.sh server-simulator server-seed

EXPOSE 8080

ENTRYPOINT ["./entrypoint.sh"]
