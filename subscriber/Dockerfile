# ── Build stage ──────────────────────────────────────────────────────────────
# One builder compiles every binary so layer cache is shared across targets.
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/subscriber \
    ./subscriber

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/logwriter \
    ./logwriter

# ── subscriber runtime ────────────────────────────────────────────────────────
FROM scratch AS subscriber

COPY --from=builder /out/subscriber /subscriber
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/subscriber"]

# ── logwriter runtime ─────────────────────────────────────────────────────────
# Two containers (logwriter-general, logwriter-alert) share this image.
# Each is given a different RABBITMQ_QUEUE + LOGWRITER_BINDING via docker-compose.
FROM scratch AS logwriter

COPY --from=builder /out/logwriter /logwriter
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/logwriter"]