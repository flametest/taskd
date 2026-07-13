# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Download deps first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# taskd uses pgx (pure-Go Postgres driver); no cgo needed, so build a static
# binary. -trimpath/-s/-w strip paths and debug info for a smaller image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/taskd ./cmd/taskd

# ---- runtime stage ----
FROM alpine:3.20

# ca-certificates: HTTPS calls to upstream tasks.
# tzdata: cron expressions resolve IANA names (e.g. Asia/Shanghai).
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/taskd /app/taskd
COPY deploy/server-config.yaml /etc/taskd/server-config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/taskd", "--config", "/etc/taskd/server-config.yaml"]
