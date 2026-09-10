FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /wishlist ./cmd/api

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /wishlist /app/wishlist
COPY --from=builder /app/migrations /app/migrations
COPY scripts/docker-entrypoint.sh /app/docker-entrypoint.sh
RUN mkdir /app/state && chown 65532:65532 /app/state && chmod 700 /app/state
USER 65532:65532

EXPOSE 8080
ENTRYPOINT ["/bin/sh", "/app/docker-entrypoint.sh"]
CMD ["/app/wishlist"]
