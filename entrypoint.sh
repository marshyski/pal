#!/bin/sh

PASS=$(tr </dev/urandom -dc 'A-Za-z0-9_!@#$%^&*()-+=' | head -c 12)
ENCRYPT=$(tr </dev/urandom -dc 'A-Za-z0-9_!@#$%^&*()-+=' | head -c 16)
SESSION=$(tr </dev/urandom -dc 'A-Za-z0-9_!@#$%^&*()-+=' | head -c 16)

cd /pal || echo "error cannot change into /pal directory"

echo

if [ "$HTTP_LISTEN" != "" ]; then
    sed -i "s|listen:.*|listen: $HTTP_LISTEN|" /etc/pal/pal.yml
fi

if [ "$HTTP_TIMEOUT_MIN" != "" ]; then
    sed -i "s|timeout:.*|timeout: $HTTP_TIMEOUT_MIN|" /etc/pal/pal.yml
fi

if [ "$HTTP_BODY_LIMIT" != "" ]; then
    sed -i "s|body_limit:.*|body_limit: $HTTP_BODY_LIMIT|" /etc/pal/pal.yml
fi

if [ "$HTTP_CORS_ALLOW_ORIGINS" != "" ]; then
    sed -i "s|cors_allow_origins:.*|cors_allow_origins: $HTTP_CORS_ALLOW_ORIGINS|" /etc/pal/pal.yml
fi

if [ "$HTTP_UI_BASIC_AUTH" != "" ]; then
    sed -i "s|basic_auth:.*|basic_auth: $HTTP_UI_BASIC_AUTH|" /etc/pal/pal.yml
else
    sed -i "s|basic_auth:.*|basic_auth: admin $PASS|" /etc/pal/pal.yml
    echo "basic_auth:      admin $PASS"
fi

if [ "$HTTP_AUTH_HEADER" != "" ]; then
    sed -i "s|auth_header:.*|auth_header: $HTTP_AUTH_HEADER|" /etc/pal/pal.yml
else
    sed -i "s|auth_header:.*|auth_header: x-pal-auth $PASS|" /etc/pal/pal.yml
    echo "auth_header:     x-pal-auth $PASS"
fi

if [ "$HTTP_SESSION_SECRET" != "" ]; then
    sed -i "s|session_secret:.*|session_secret: $HTTP_SESSION_SECRET|" /etc/pal/pal.yml
else
    sed -i "s|session_secret:.*|session_secret: $SESSION|" /etc/pal/pal.yml
    echo "session_secret:  $SESSION"
fi

if [ "$DB_ENCRYPT_KEY" != "" ]; then
    sed -i "s|encrypt_key:.*|encrypt_key: $DB_ENCRYPT_KEY|" /etc/pal/pal.yml
else
    sed -i "s|encrypt_key:.*|encrypt_key: $ENCRYPT|" /etc/pal/pal.yml
    echo "encrypt_key:     $ENCRYPT"
fi

if [ "$GLOBAL_DEBUG" != "" ]; then
    sed -i "s|debug:.*|debug: $GLOBAL_DEBUG|" /etc/pal/pal.yml
fi

echo

./pal -c /etc/pal/pal.yml -d /etc/pal/actions
