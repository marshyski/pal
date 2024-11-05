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

COPY ./pal /pal/
COPY ./pal.yml ./entrypoint.sh ./localhost.key ./localhost.pem /etc/pal/
COPY ./test/*.yml /etc/pal/actions/

WORKDIR /pal

RUN mkdir -p /etc/pal/pal.db /pal/upload && \
    sed -i "s|listen:.*|listen: 0.0.0.0:8443|" /etc/pal/pal.yml && \
    sed -i "s|  key:.*|  key: /etc/pal/localhost.key|" /etc/pal/pal.yml && \
    sed -i "s|cert:.*|cert: /etc/pal/localhost.pem|" /etc/pal/pal.yml && \
    sed -i "s|working_dir:.*|working_dir: /pal|" /etc/pal/pal.yml && \
    sed -i "s|upload_dir:.*|upload_dir: /pal/upload|" /etc/pal/pal.yml && \
    sed -i "s|path:.*|path: /etc/pal/pal.db|" /etc/pal/pal.yml && \
    sed -i "s|debug:.*|debug: false|" /etc/pal/pal.yml

RUN addgroup --gid 101010 --system pal && \
    adduser --uid 101010 --system --ingroup pal --home /pal --shell /sbin/nologin --comment "pal Service Account" pal && \
    chown -Rf pal:pal /pal /etc/pal

USER pal

EXPOSE 8443

CMD ["/etc/pal/entrypoint.sh"]
