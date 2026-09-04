FROM alpine:3.24
WORKDIR /opt/kelvin

RUN apk --no-cache add ca-certificates tzdata && update-ca-certificates \
	&& mkdir -p /etc/opt/kelvin
COPY kelvin /opt/kelvin/
COPY gui /opt/kelvin/gui

# Exec form: kelvin runs as PID 1 and receives SIGTERM on docker stop.
# Logs go to stdout only — the container runtime captures them (#138).
# Persist the configuration by mounting /etc/opt/kelvin (see README).
ENTRYPOINT ["/opt/kelvin/kelvin", "-enableUpdates=false", "-enableWebInterface=true", "-listenAddress=0.0.0.0", "-configuration=/etc/opt/kelvin/config.json"]
