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

COPY pal pal.yml ./entrypoint.sh ./test/pal-defs.yml localhost.key localhost.pem /pal/

WORKDIR /pal

EXPOSE 8443

CMD ["/pal/entrypoint.sh"]
