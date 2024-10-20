#!/bin/sh

APP="pal"
INSTALL_DIR="/$APP"
UPLOAD_DIR="/$APP/upload"
ACTIONS_DIR="/$APP/actions"

ID="101010"

if [ "$(id $ID 2>/dev/null)" = "" ]; then
    addgroup --gid "$ID" --system "$APP"
    adduser --uid "$ID" --system --ingroup "$APP" --home "$INSTALL_DIR" --shell /sbin/nologin --comment "pal Service Account" "$APP"
    chmod -f 0700 "$INSTALL_DIR"
fi

if [ ! -d "$INSTALL_DIR" ]; then
    mkdir -p "$INSTALL_DIR"
    chmod -f 0700 "$INSTALL_DIR"
    chown -f "$APP":"$APP" "$INSTALL_DIR"
fi

if [ ! -d "$UPLOAD_DIR" ]; then
    mkdir -p "$UPLOAD_DIR"
    chmod -f 0700 "$UPLOAD_DIR"
    chown -f "$APP":"$APP" "$UPLOAD_DIR"
fi

if [ ! -d "$ACTIONS_DIR" ]; then
    mkdir -p "$ACTIONS_DIR"
    chmod -f 0700 "$ACTIONS_DIR"
    chown -f "$APP":"$APP" "$ACTIONS_DIR"
fi
