FROM docker.io/library/debian:stable-slim@sha256:4448d44b91bf4a13cb1b4e02d9d5f87ed40621d6e33f0ae7b6ddf71d57e29364

COPY ./pal /pal/
COPY ./entrypoint.sh ./localhost.key ./localhost.pem /etc/pal/

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get dist-upgrade -y && \
    apt-get install -y --no-install-recommends \
        curl \
        tzdata \
        ca-certificates \
        adduser \
        jq && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /etc/pal/pal.db /etc/pal/actions /pal/upload && \
    addgroup --gid 1000 --system pal && \
    adduser --uid 1000 --system --ingroup pal --home /pal --shell /sbin/nologin pal && \
    chown -Rf pal:pal /pal /etc/pal && \
    chmod -f 0755 /etc/pal/*.sh /pal/pal

USER pal

WORKDIR /pal

EXPOSE 8443

ENTRYPOINT ["/etc/pal/entrypoint.sh"]
