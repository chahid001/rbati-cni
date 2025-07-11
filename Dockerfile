FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /rbati-agent ./cmd/agent/main.go

FROM alpine:3.20

RUN apk add --no-cache iptables iproute2
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /rbati-agent /usr/local/bin/rbati-agent

USER appuser

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD pgrep rbati-agent || exit 1

ENTRYPOINT ["rbati-agent"]