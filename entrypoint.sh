#!/bin/sh

cd /pal || echo "error cannot change into /pal directory"

if [ "$HTTP_LISTEN" != "" ]; then
    sed -i "s|listen:.*|listen: $HTTP_LISTEN|" /pal/pal.yml
fi

if [ "$HTTP_TIMEOUT_MIN" != "" ]; then
    sed -i "s|timeout:.*|timeout: $HTTP_TIMEOUT_MIN|" /pal/pal.yml
fi

if [ "$HTTP_BODY_LIMIT" != "" ]; then
    sed -i "s|body_limit:.*|body_limit: $HTTP_BODY_LIMIT|" /pal/pal.yml
fi

if [ "$HTTP_CORS_ALLOW_ORIGINS" != "" ]; then
    sed -i "s|cors_allow_origins:.*|cors_allow_origins: $HTTP_CORS_ALLOW_ORIGINS|" /pal/pal.yml
fi

if [ "$HTTP_UI_BASIC_AUTH" != "" ]; then
    sed -i "s|basic_auth:.*|basic_auth: $HTTP_UI_BASIC_AUTH|" /pal/pal.yml
fi

if [ "$DB_ENCRYPT_KEY" != "" ]; then
    sed -i "s|encrypt_key:.*|encrypt_key: $DB_ENCRYPT_KEY|" /pal/pal.yml
fi

if [ "$DB_AUTH_HEADER" != "" ]; then
    sed -i "s|auth_header:.*|auth_header: $DB_AUTH_HEADER|" /pal/pal.yml
fi

./pal
