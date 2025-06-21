#!/bin/bash

HOST='127.0.0.1'
PORT='8443'
HEADER='X-Pal-Auth: PaLLy!@#890-'
BASIC_AUTH='pal:p@LLy5'

while [[ $# -gt 0 ]]; do
    case "$1" in
    -port)
        PORT="$2"
        shift 2
        ;;
    -host)
        HOST="$2"
        shift 2
        ;;
    -header)
        HEADER="$2"
        shift 2
        ;;
    -basicauth)
        BASIC_AUTH="$2"
        shift 2
        ;;
    *)
        echo "Unknown option: $1" >&2
        exit 1
        ;;
    esac
done

URL="https://$HOST:$PORT"
HEALTHCHECK="$URL/v1/pal/health"

echo "URL=$URL"
echo "HEALTHCHECK=$HEALTHCHECK"
echo "HEADER=$HEADER"
echo "BASIC_AUTH=$BASIC_AUTH"

# health_check
OUT=$(curl -sSk "$HEALTHCHECK")

if [ "$(echo "$OUT" | grep -c "ok")" = 1 ]; then
    echo "[pass] health_check"
else
    echo "$OUT"
    echo "[fail] health_check" && exit 1
fi

# save cookie
curl -sSk -XPOST -d "username=$(echo "$BASIC_AUTH" | awk -F':' '{ print $1 }')" -d "password=$(echo "$BASIC_AUTH" | awk -F':' '{ print $2 }')" --cookie-jar ./pal.cookie "$URL/v1/pal/ui/login" 1>/dev/null

# file_upload
echo 'test' >./test.txt
OUT=$(curl -sSk -XPOST -F files='@./test.txt' -b ./pal.cookie "$URL/v1/pal/ui/files/upload")
if [ "$(echo "$OUT" | grep -c "uploaded 1 file")" = 1 ]; then
    echo "[pass] file_upload"
else
    echo "$OUT"
    rm -f ./pal.cookie ./test.txt
    echo "[fail] file_upload" && exit 1
fi
rm -f ./test.txt

# file_download
OUT=$(curl -sSk -b ./pal.cookie "$URL/v1/pal/ui/files/download/test.txt")
if [ "$(echo "$OUT" | grep -c "test")" = 1 ]; then
    echo "[pass] file_download"
else
    echo "$OUT"
    rm -f ./pal.cookie
    echo "[fail] file_download" && exit 1
fi

# file_delete
OUT=$(curl -sSk -b ./pal.cookie "$URL/v1/pal/ui/files/delete/test.txt")
if [ "$OUT" = "" ]; then
    echo "[pass] file_delete"
else
    echo "$OUT"
    rm -f ./pal.cookie
    echo "[fail] file_delete" && exit 1
fi
rm -f ./pal.cookie

# auth
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/auth?input=helloworld")

if [ "$(echo "$OUT" | grep -c "helloworld")" = 1 ]; then
    echo "[pass] auth"
else
    echo "$OUT"
    echo "[fail] auth" && exit 1
fi

# no_auth
OUT=$(curl -sSk "$URL/v1/pal/run/test/no_auth")

if [ "$(echo "$OUT" | grep -c "no_auth")" = 1 ]; then
    echo "[pass] no_auth"
else
    echo "$OUT"
    echo "[fail] no_auth" && exit 1
fi

# fail
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/fail")

if [ "$(echo "$OUT" | grep -c "exit")" = 1 ]; then
    echo "[pass] fail"
else
    echo "$OUT"
    echo "[fail] fail" && exit 1
fi

# fail_timeout
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/fail_timeout")

if [ "$(echo "$OUT" | grep -c "killed")" = 1 ]; then
    echo "[pass] fail_timeout"
else
    echo "$OUT"
    echo "[fail] fail_timeout" && exit 1
fi

# block
curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/block" 1>/dev/null &
sleep 1
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/block?input=1")

if [ "$(echo "$OUT" | grep -c "block")" = 1 ]; then
    echo "[pass] block"
else
    echo "$OUT"
    echo "[fail] block" && exit 1
fi

# no_block
curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/no_block" 1>/dev/null &
sleep 1
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/no_block")

if [ "$(echo "$OUT" | grep -c "default test no_block")" = 1 ]; then
    echo "[pass] no_block"
else
    echo "$OUT"
    echo "[fail] no_block" && exit 1
fi

# no_output
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/no_output")

if [ "$(echo "$OUT" | grep -c "done")" = 1 ]; then
    echo "[pass] no_output"
else
    echo "$OUT"
    echo "[fail] no_output" && exit 1
fi

# input_validate
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/input_validate?input=123")

if [ "$(echo "$OUT" | grep -c "input_validate")" = 1 ]; then
    echo "[pass] input_validate"
else
    echo "$OUT"
    echo "[fail] input_validate" && exit 1
fi

# retry_fail
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/retry_fail?input=123")

if [ "$(echo "$OUT" | grep -c "2 retries")" = 1 ]; then
    echo "[pass] retry_fail"
else
    echo "$OUT"
    echo "[fail] retry_fail" && exit 1
fi

# json/parse
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/json/parse?input=%7B%22hello%22%3A%22world%22%7D")
if [ "$(echo "$OUT" | grep -c "hello")" = 1 ]; then
    echo "[pass] json/parse"
else
    echo "$OUT"
    echo "[fail] json/parse" && exit 1
fi

# invalid
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/invalid")
if [ "$(echo "$OUT" | grep -c "invalid")" = 1 ]; then
    echo "[pass] invalid"
else
    echo "$OUT"
    echo "[fail] invalid" && exit 1
fi

# html/index_cache
OUT=$(curl -sSk -H "$HEADER" -o /dev/null -w '%{content_type}' "$URL/v1/pal/run/html/index_cache")
if [ "$(echo "$OUT" | grep -c 'text/html')" = 1 ]; then
    echo "[pass] html/index_cache"
else
    echo "$OUT"
    echo "[fail] html/index_cache" && exit 1
fi

# DB PUT/GET
curl -sSk -XPUT -u "$BASIC_AUTH" -d 'UniqString123' "$URL/v1/pal/db/put?key=test" 1>/dev/null
OUT=$(curl -sSk -u "$BASIC_AUTH" "$URL/v1/pal/db/get?key=test")
if [ "$(echo "$OUT" | grep -c "UniqString123")" = 1 ]; then
    echo "[pass] db/put/get"
else
    echo "$OUT"
    echo "[fail] db/put/get" && exit 1
fi

# DB Delete
curl -sfk -XDELETE -u "$BASIC_AUTH" "$URL/v1/pal/db/delete?key=test" 1>/dev/null
OUT=$(curl -sSk -u "$BASIC_AUTH" "$URL/v1/pal/db/get?key=test")
if [ "$(echo "$OUT" | grep -c "value not found")" = 1 ]; then
    echo "[pass] db/delete"
else
    echo "$OUT"
    echo "[fail] db/delete" && exit 1
fi

# GET Crons
OUT=$(curl -sSk -u "$BASIC_AUTH" "$URL/v1/pal/crons")
if [ "$(echo "$OUT" | grep -c "no_auth")" -ge 1 ]; then
    echo "[pass] crons/get"
else
    echo "$OUT"
    echo "[fail] crons/get" && exit 1
fi

# GET Notifications
curl -sSk -u "$BASIC_AUTH" \
    -d '{"notification":"THE QUICK BROWN FOX JUMPED OVER THE LAZY DOGS BACK 1234567890","group":"json"}' \
    -H "content-type: application/json" -XPUT "$URL/v1/pal/notifications" 1>/dev/null
OUT=$(curl -sSk -u "$BASIC_AUTH" "$URL/v1/pal/notifications")
if [ "$(echo "$OUT" | grep -c "1234567890")" -ge 1 ]; then
    echo "[pass] notifications/get"
else
    echo "$OUT"
    echo "[fail] notifications/get" && exit 1
fi
