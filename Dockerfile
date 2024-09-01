FROM debian:stable-slim

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get dist-upgrade -y && \
    apt-get install -y --no-install-recommends \
        curl \
        ca-certificates \
        jq && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY pal pal.yml ./test/pal-defs.yml localhost.key localhost.pem /pal/

RUN useradd -m -s /bin/bash --uid 9000 --home /pal pal && \
    mkdir -p /pal/upload && \
    chown -R pal:pal /pal && \
    chmod 0700 /pal

USER pal

WORKDIR /pal

EXPOSE 8443

CMD ["./pal"]
