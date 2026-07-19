FROM docker.io/library/debian:stable-slim@sha256:ee12ffb55625b99d62837a72f037d9b2f18fd0c787a89c2b9a4f09666c48776c

COPY ./pal /usr/bin/
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
    chown -Rf pal:pal /etc/pal && \
    chown -Rf pal:pal /etc/pal/pal.db /pal && \
    chmod -f 0640 /etc/pal/localhost.* && \
    chmod -f 0755 /usr/bin/pal /etc/pal/*.sh

USER pal

WORKDIR /pal

EXPOSE 8443

HEALTHCHECK --interval=60s --timeout=15s --retries=2 \
  CMD /usr/bin/pal -s -c /etc/pal/pal.yml || exit 1

ENTRYPOINT ["/etc/pal/entrypoint.sh"]
