FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/.env /app/.env

# Create logs directory
RUN mkdir -p logs

EXPOSE 8080 50051

ENTRYPOINT ["/app/server"]
