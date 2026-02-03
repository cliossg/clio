# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/clio \
    .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/clio /app/clio

RUN mkdir -p /app/data/db /app/data/sites

EXPOSE 8080 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ping || exit 1

ENTRYPOINT ["/app/clio"]
