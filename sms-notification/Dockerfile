FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service ./cmd/main.go

# Final stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy the binary from builder
COPY --chown=nonroot:nonroot --from=builder /app/notification-service .

USER nonroot:nonroot

# Run the binary
CMD ["./notification-service"]
