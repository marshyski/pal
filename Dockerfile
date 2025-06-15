FROM debian:stable-slim@sha256:50db38a20a279ccf50761943c36f9e82378f92ef512293e1239b26bb77a8b496

COPY ./pal /pal/
COPY ./entrypoint.sh ./localhost.key ./localhost.pem /etc/pal/

WORKDIR /pal

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get dist-upgrade -y && \
    apt-get install -y --no-install-recommends \
        curl \
        tzdata \
        ca-certificates \
        jq && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /etc/pal/pal.db /etc/pal/actions /pal/upload && \
    addgroup --gid 101010 --system pal && \
    adduser --uid 101010 --system --ingroup pal --home /pal --shell /sbin/nologin pal && \
    chown -Rf pal:pal /pal /etc/pal

USER pal

EXPOSE 8443

ENTRYPOINT ["/etc/pal/entrypoint.sh"]
