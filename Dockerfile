FROM golang:1.25rc1-alpine3.22 AS build

RUN apk add --no-cache bash curl git tar

WORKDIR /src
COPY . /src

RUN go build -o bin/rbati-agent cmd/controller/main.go

FROM alpine:3.22

RUN apk add --no-cache iptables iproute2

COPY --from=build /src/bin/rbati-agent /rbati-agent

ENTRYPOINT ["/rbati-agent"]
