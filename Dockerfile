FROM alpine:3.22

COPY bin/rbati-agent /rbati-agent

ENTRYPOINT ["/rbati-agent"]
