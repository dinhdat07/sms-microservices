FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o reporting-service ./cmd/reporting/main.go

# Final stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy the binary from builder
COPY --chown=nonroot:nonroot --from=builder /app/reporting-service .

# Copy email templates
COPY --chown=nonroot:nonroot --from=builder /app/internal/service/templates ./internal/service/templates

USER nonroot:nonroot

# Expose ports (HTTP API and gRPC)
EXPOSE 8002 50053

# Run the binary
CMD ["./reporting-service"]
