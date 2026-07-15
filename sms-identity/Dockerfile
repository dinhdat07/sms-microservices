FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/server/main.go

# Create logs dir in builder so we can copy it with correct permissions
RUN mkdir -p /app/logs

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy the binary and logs dir with nonroot ownership
COPY --chown=nonroot:nonroot --from=builder /app/bin/server /app/server
COPY --chown=nonroot:nonroot --from=builder /app/logs /app/logs

USER nonroot:nonroot

EXPOSE 8080 50051

ENTRYPOINT ["/app/server"]
