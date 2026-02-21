#!/bin/sh

APP="pal"
INSTALL_DIR="/$APP"
UPLOAD_DIR="/$APP/upload"
CONF_DIR="/etc/$APP"
ACTIONS_DIR="/etc/$APP/actions"
DB_DIR="/etc/$APP/$APP.db"

ID="101010"

if [ "$(id $ID 2>/dev/null)" = "" ]; then
    addgroup --gid "$ID" --system "$APP" || groupadd --gid "$ID" --system "$APP"
    adduser --uid "$ID" --system --ingroup "$APP" --home "$INSTALL_DIR" --shell /sbin/nologin "$APP" || adduser --uid "$ID" --system --gid "$ID" --home-dir "$INSTALL_DIR" --shell /sbin/nologin "$APP"
    chmod -f 0750 "$INSTALL_DIR"
fi

if [ ! -d "$INSTALL_DIR" ]; then
    mkdir -p "$INSTALL_DIR"
    chmod -f 0750 "$INSTALL_DIR"
    chown -f "$APP":"$APP" "$INSTALL_DIR"
fi

if [ ! -d "$UPLOAD_DIR" ]; then
    mkdir -p "$UPLOAD_DIR"
    chmod -f 0750 "$UPLOAD_DIR"
    chown -f "$APP":"$APP" "$UPLOAD_DIR"
fi

if [ ! -d "$CONF_DIR" ]; then
    mkdir -p "$CONF_DIR"
    chmod -f 0750 "$CONF_DIR"
    chown -f root:"$APP" "$CONF_DIR"
fi

if [ ! -d "$ACTIONS_DIR" ]; then
    mkdir -p "$ACTIONS_DIR"
    chmod -f 0750 "$ACTIONS_DIR"
    chown -f root:"$APP" "$ACTIONS_DIR"
fi

if [ ! -d "$DB_DIR" ]; then
    mkdir -p "$DB_DIR"
    chmod -f 0750 "$DB_DIR"
    chown -f "$APP":"$APP" "$DB_DIR"
fi

if ! ls /etc/pal/*.pem /etc/pal/*.key >/dev/null 2>&1; then
  openssl req -x509 -newkey rsa:4096 -nodes \
    -keyout /etc/pal/localhost.key -out /etc/pal/localhost.pem \
    -days 365 -sha256 \
    -subj '/CN=localhost' \
    -addext "subjectAltName=IP:127.0.0.1,DNS:localhost"

    chown -f root:pal /etc/pal/localhost.*
    chmod -f 0640 /etc/pal/localhost.*
fi
