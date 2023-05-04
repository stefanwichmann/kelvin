FROM alpine:latest
WORKDIR /opt/kelvin
VOLUME /etc/opt/kelvin

RUN apk --no-cache add ca-certificates tzdata && update-ca-certificates
COPY kelvin /opt/kelvin/
COPY gui /opt/kelvin/gui

ENTRYPOINT /opt/kelvin/kelvin -enableUpdates=false -enableWebInterface=true -configuration=/etc/opt/kelvin/config.json 2>&1 | tee /var/log/kelvin.log
