# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Install necessary Go tools
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
RUN	go install github.com/pressly/goose/v3/cmd/goose@latest
RUN	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest



# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /docko ./cmd/server

# Production stage
FROM alpine:3.21

# Install thumbnail tools:
# - poppler-utils: provides pdftoppm for PDF to PNG conversion
# - libwebp-tools: provides cwebp for PNG to WebP conversion
RUN apk add --no-cache \
    ca-certificates \
    poppler-utils \
    libwebp-tools

WORKDIR /app

# Copy binary from builder
COPY --from=builder /docko /app/docko

# Copy static assets and templates
COPY static /app/static
COPY assets /app/assets

# Create storage directory
RUN mkdir -p /app/storage

EXPOSE 3000

CMD ["/app/docko"]
