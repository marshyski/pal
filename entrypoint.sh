#!/bin/sh

PASS=$(tr </dev/urandom -dc 'A-Za-z0-9_!@#$%^&*()-+=' | head -c 12)
ENCRYPT=$(tr </dev/urandom -dc 'A-Za-z0-9_-' | head -c 16)
SESSION=$(tr </dev/urandom -dc 'A-Za-z0-9_!@#$%^&*()-+=' | head -c 16)

cd /pal || echo "error cannot change into /pal directory"

echo

if [ ! -f "/etc/pal/pal.yml" ]; then
    if [ "$GLOBAL_DEBUG" = "" ]; then
        GLOBAL_DEBUG="false"
    fi

    if [ "$GLOBAL_TIMEZONE" = "" ]; then
        GLOBAL_TIMEZONE="UTC"
    fi

    if [ "$GLOBAL_CMD_PREFIX" = "" ]; then
        GLOBAL_CMD_PREFIX="/bin/sh -c"
    fi

    if [ "$GLOBAL_WORKDIR" = "" ]; then
        GLOBAL_WORKDIR="/pal"
    fi

    if [ "$HTTP_LISTEN" = "" ]; then
        HTTP_LISTEN="0.0.0.0:8443"
    fi

    if [ "$HTTP_TIMEOUT_MIN" = "" ]; then
        HTTP_TIMEOUT_MIN="10"
    fi

    if [ "$HTTP_BODY_LIMIT" = "" ]; then
        HTTP_BODY_LIMIT="90M"
    fi

    if [ "$HTTP_MAX_AGE" = "" ]; then
        HTTP_MAX_AGE="3600"
    fi

    if [ "$HTTP_CORS_ALLOW_ORIGINS" = "" ]; then
        HTTP_CORS_ALLOW_ORIGINS="[]"
    fi

    if [ "$HTTP_UI_BASIC_AUTH" = "" ]; then
        HTTP_UI_BASIC_AUTH="pal $PASS"
        echo "basic_auth:      pal $PASS"
    fi

    if [ "$HTTP_UI_UPLOAD_DIR" = "" ]; then
        HTTP_UI_UPLOAD_DIR="/pal/upload"
    fi

    if [ "$HTTP_AUTH_HEADER" = "" ]; then
        HTTP_AUTH_HEADER="X-Pal-Auth $PASS"
        echo "auth_header:     X-Pal-Auth $PASS"
    fi

    if [ "$HTTP_SESSION_SECRET" = "" ]; then
        HTTP_SESSION_SECRET="$SESSION"
        echo "session_secret:  $SESSION"
    fi

    if [ "$HTTP_PROMETHEUS" = "" ]; then
        HTTP_PROMETHEUS="false"
    fi

    if [ "$DB_ENCRYPT_KEY" = "" ]; then
        DB_ENCRYPT_KEY="$ENCRYPT"
        echo "encrypt_key:     $ENCRYPT"
    fi

    if [ "$DB_PATH" = "" ]; then
        DB_PATH="/etc/pal/pal.db"
    fi

    if [ "$NOTIFICATIONS_MAX" = "" ]; then
        NOTIFICATIONS_MAX="100"
    fi
    mkdir -p /etc/pal/pal.db
    cat <<EOF >/etc/pal/pal.yml
global:
  timezone: $GLOBAL_TIMEZONE
  cmd_prefix: $GLOBAL_CMD_PREFIX
  working_dir: $GLOBAL_WORKDIR
  debug: $GLOBAL_DEBUG
http:
  listen: $HTTP_LISTEN
  timeout_min: $HTTP_TIMEOUT_MIN
  body_limit: $HTTP_BODY_LIMIT
  max_age: $HTTP_MAX_AGE
  key: "/etc/pal/localhost.key"
  cert: "/etc/pal/localhost.pem"
  cors_allow_origins: $HTTP_CORS_ALLOW_ORIGINS
  session_secret: $HTTP_SESSION_SECRET
  auth_header: $HTTP_AUTH_HEADER
  prometheus: $HTTP_PROMETHEUS
  ui:
    upload_dir: $HTTP_UI_UPLOAD_DIR
    basic_auth: $HTTP_UI_BASIC_AUTH
db:
  encrypt_key: $DB_ENCRYPT_KEY
  path: $DB_PATH
notifications:
  max: $NOTIFICATIONS_MAX
EOF
fi

echo

./pal -c /etc/pal/pal.yml -d /etc/pal/actions
