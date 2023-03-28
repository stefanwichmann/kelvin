FROM alpine:latest
WORKDIR /opt/kelvin
VOLUME /etc/opt/kelvin

RUN apk --no-cache add ca-certificates tzdata && update-ca-certificates
COPY kelvin /opt/kelvin/

ENTRYPOINT /opt/kelvin/kelvin -enableWebInterface -configuration=/etc/opt/kelvin/config.json 2>&1 | tee /var/log/kelvin.log
