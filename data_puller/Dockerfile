FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install required system packages
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o data_puller ./cmd/main.go

# Create migrations directory in the final location
RUN mkdir -p /app/migrations

# Copy migrations to the correct location
RUN cp internal/database/migrations/* /app/migrations/

# Final stage
FROM alpine:3.19

WORKDIR /app

# Copy binary and migrations from builder
COPY --from=builder /app/data_puller .
COPY --from=builder /app/migrations ./migrations

# Create non-root user
RUN adduser -D appuser
# Give permissions to migrations directory
RUN chown -R appuser:appuser /app
USER appuser

CMD ["./data_puller"]