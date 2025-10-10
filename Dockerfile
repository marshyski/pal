FROM debian:stable-slim@sha256:d6743b7859c917a488ca39f4ab5e174011305f50b44ce32d3b9ea5d81b291b3b

WORKDIR /pal

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get dist-upgrade -y && \
    apt-get install -y --no-install-recommends \
        curl \
        tzdata \
        ca-certificates \
        jq \
        libgpgme11 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /etc/pal/pal.db /etc/pal/actions /pal/upload && \
    addgroup --gid 1000 --system pal && \
    adduser --uid 1000 --system --ingroup pal --home /pal --shell /sbin/nologin pal

COPY ./pal /pal/
COPY ./entrypoint.sh ./localhost.key ./localhost.pem /etc/pal/

RUN chown pal:pal /pal /etc/pal && \
    chown -R pal:pal /pal /etc/pal && \
    chmod 0755 /etc/pal/entrypoint.sh /pal/pal

USER pal

EXPOSE 8443

ENTRYPOINT ["/etc/pal/entrypoint.sh"]
