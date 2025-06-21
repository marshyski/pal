#!/bin/sh

# Seed test data using default configuration values

AUTH_HEADER='X-Pal-Auth: PaLLy!@#890-'

# Create Notification
curl -ks -H "$AUTH_HEADER" \
    -d '{"notification":"THE QUICK BROWN FOX JUMPED OVER THE LAZY DOGS BACK 1234567890","group":"json"}' \
    -H "content-type: application/json" -XPUT \
    'https://127.0.0.1:8443/v1/pal/notifications'
