#!/bin/sh

PASS=$(tr </dev/urandom -dc 'A-Za-z0-9_!@#$%^&*()-+=' | head -c 12)
ENCRYPT=$(tr </dev/urandom -dc 'A-Za-z0-9_-' | head -c 16)
SESSION=$(tr </dev/urandom -dc 'A-Za-z0-9_!@#$%^&*()-+=' | head -c 16)

cd /pal || echo "error cannot change into /pal directory"

echo

if [ ! -f "/etc/pal/pal.yml" ]; then
    if [ "$HTTP_LISTEN" = "" ]; then
        HTTP_LISTEN="0.0.0.0:8443"
    fi

    if [ "$HTTP_TIMEOUT_MIN" = "" ]; then
        HTTP_TIMEOUT_MIN="10"
    fi

    if [ "$HTTP_BODY_LIMIT" = "" ]; then
        HTTP_BODY_LIMIT="90M"
    fi

    if [ "$HTTP_CORS_ALLOW_ORIGINS" = "" ]; then
        HTTP_CORS_ALLOW_ORIGINS="[]"
    fi

    if [ "$HTTP_UI_BASIC_AUTH" = "" ]; then
        HTTP_UI_BASIC_AUTH="$PASS"
        echo "basic_auth:      admin $PASS"
    fi

    if [ "$HTTP_AUTH_HEADER" = "" ]; then
        HTTP_AUTH_HEADER="x-pal-auth $PASS"
        echo "auth_header:     x-pal-auth $PASS"
    fi

    if [ "$HTTP_SESSION_SECRET" = "" ]; then
        HTTP_SESSION_SECRET="$SESSION"
        echo "session_secret:  $SESSION"
    fi

    if [ "$DB_ENCRYPT_KEY" = "" ]; then
        DB_ENCRYPT_KEY="$ENCRYPT"
        echo "encrypt_key:     $ENCRYPT"
    fi
    mkdir -p /etc/pal/pal.db
    cat <<EOF >/etc/pal/pal.yml
global:
  cmd_prefix: "/bin/sh -c"
  working_dir: /pal
http:
  listen: $HTTP_LISTEN
  timeout_min: $HTTP_TIMEOUT_MIN
  body_limit: $HTTP_BODY_LIMIT
  key: "/etc/pal/localhost.key"
  cert: "/etc/pal/localhost.pem"
  cors_allow_origins: $HTTP_CORS_ALLOW_ORIGINS
  session_secret: $HTTP_SESSION_SECRET
  auth_header: $HTTP_AUTH_HEADER
  ui:
    upload_dir: /pal/upload
    basic_auth: $HTTP_UI_BASIC_AUTH
db:
  encrypt_key: $DB_ENCRYPT_KEY
  path: "/etc/pal/pal.db"
notifications:
  max: 100
EOF
fi

echo

./pal -c /etc/pal/pal.yml -d /etc/pal/actions
