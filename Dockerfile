FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /wishlist ./cmd/api

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /wishlist /app/wishlist
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080
CMD ["/app/wishlist"]