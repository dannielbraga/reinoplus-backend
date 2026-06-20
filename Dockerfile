FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates openssl

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY migrations /app/migrations

RUN mkdir -p /app/keys \
  && openssl genrsa -out /app/keys/private.pem 2048 \
  && openssl rsa -in /app/keys/private.pem -pubout -out /app/keys/public.pem

ENV SERVER_PORT=8080
ENV JWT_PRIVATE_KEY_PATH=/app/keys/private.pem
ENV JWT_PUBLIC_KEY_PATH=/app/keys/public.pem
ENV JWT_ACCESS_TTL=15m
ENV JWT_REFRESH_TTL=168h
ENV CORS_ALLOWED_ORIGINS=*

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
  CMD wget -qO- http://127.0.0.1:8080/health || exit 1

CMD ["/app/api"]
