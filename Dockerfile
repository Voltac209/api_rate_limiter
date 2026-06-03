FROM golang:1.25-alpine AS builder
    WORKDIR /app

    COPY go.mod go.sum ./
    RUN go mod download

    COPY . .
    RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/ratelimiter ./server.go

    FROM alpine:3.20
    WORKDIR /app
    COPY --from=builder /bin/ratelimiter /usr/local/bin/ratelimiter

    EXPOSE 50051
    ENV LIMITER_BACKEND=inmemory
    CMD ["ratelimiter"]