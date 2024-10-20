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

COPY pal pal.yml ./entrypoint.sh localhost.key localhost.pem /pal/
COPY ./test/*.yml /pal/actions/

RUN sed -i "s|listen:.*|listen: 0.0.0.0:8443|" /pal/pal.yml && \
    sed -i "s|  key:.*|  key: /pal/localhost.key|" /pal/pal.yml && \
    sed -i "s|cert:.*|cert: /pal/localhost.pem|" /pal/pal.yml && \
    sed -i "s|upload_dir:.*|upload_dir: /pal/upload|" /pal/pal.yml && \
    sed -i "s|path:.*|path: /pal/pal.db|" /pal/pal.yml && \
    sed -i "s|debug:.*|debug: false|" /pal/pal.yml && \
    mkdir -p /pal/pal.db /pal/upload

WORKDIR /pal

RUN addgroup --gid 101010 --system pal && \
    adduser --uid 101010 --system --ingroup pal --home /pal --shell /sbin/nologin --comment "pal Service Account" pal && \
    chown -Rf pal:pal /pal

USER pal

EXPOSE 8443

CMD ["/pal/entrypoint.sh"]
