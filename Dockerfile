FROM golang:1.25rc1-alpine3.22 AS builder

RUN apk add --no-cache bash git

WORKDIR /app

COPY . /app

RUN go build -o bin/rbati-agent cmd/agent/main.go

FROM alpine:3.20

RUN apk add --no-cache iptables iproute2 && \
  addgroup -S rbati-agent-group && adduser -S rbati-agent-user -G rbati-agent-group

COPY --from=builder /app/bin/rbati-agent /usr/local/bin/rbati-agent

USER rbati-agent-user

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD pgrep rbati-agent || exit 1

ENTRYPOINT ["rbati-agent"]