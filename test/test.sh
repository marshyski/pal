#!/bin/sh

HOST='127.0.0.1'
PORT='8443'
HEADER='X-Pal-Auth: PaLLy!@#890-'
BASIC_AUTH='pal:p@LLy5'
COOKIE_FILE="./pal.cookie"
TEST_FILE="./test.txt"

cleanup() {
    echo "Cleaning up temporary files..."
    rm -f "$COOKIE_FILE" "$TEST_FILE"
}

# Trap signals: runs cleanup on exit (success or fail) and interrupts (SIGINT/SIGTERM)
trap cleanup EXIT

contains() {
    case "$1" in
        *"$2"*) return 0 ;;
        *) return 1 ;;
    esac
}

while [ $# -gt 0 ]; do
    case "$1" in
    -port) PORT="$2"; shift 2 ;;
    -host) HOST="$2"; shift 2 ;;
    -header) HEADER="$2"; shift 2 ;;
    -basicauth) BASIC_AUTH="$2"; shift 2 ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
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
if contains "$OUT" "ok"; then
    echo "[pass] health_check"
else
    echo "$OUT"
    echo "[fail] health_check" && exit 1
fi

# save cookie
USER=$(echo "$BASIC_AUTH" | cut -d':' -f1)
PASS=$(echo "$BASIC_AUTH" | cut -d':' -f2)
curl -sSk -XPOST -d "username=$USER" -d "password=$PASS" --cookie-jar "$COOKIE_FILE" "$URL/v1/pal/ui/login" >/dev/null

# file_upload
echo 'test' > "$TEST_FILE"
OUT=$(curl -sSk -XPOST -F files="@$TEST_FILE" -b "$COOKIE_FILE" "$URL/v1/pal/ui/files/upload")
if contains "$OUT" "uploaded 1 file"; then
    echo "[pass] file_upload"
else
    echo "$OUT"
    echo "[fail] file_upload" && exit 1
fi

# file_download
OUT=$(curl -sSk -b "$COOKIE_FILE" "$URL/v1/pal/ui/files/download/test.txt")
if contains "$OUT" "test"; then
    echo "[pass] file_download"
else
    echo "$OUT"
    echo "[fail] file_download" && exit 1
fi

# file_delete
OUT=$(curl -sSk -b "$COOKIE_FILE" "$URL/v1/pal/ui/files/delete/test.txt")
if [ -z "$OUT" ]; then
    echo "[pass] file_delete"
else
    echo "$OUT"
    echo "[fail] file_delete" && exit 1
fi

# auth
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/auth?input=helloworld")
if contains "$OUT" "helloworld"; then
    echo "[pass] auth"
else
    echo "$OUT"
    echo "[fail] auth" && exit 1
fi

# no_auth
OUT=$(curl -sSk "$URL/v1/pal/run/test/no_auth")
if contains "$OUT" "no_auth"; then
    echo "[pass] no_auth"
else
    echo "$OUT"
    echo "[fail] no_auth" && exit 1
fi

# fail
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/fail")
if contains "$OUT" "exit"; then
    echo "[pass] fail"
else
    echo "$OUT"
    echo "[fail] fail" && exit 1
fi

# fail_timeout
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/fail_timeout")
if contains "$OUT" "killed"; then
    echo "[pass] fail_timeout"
else
    echo "$OUT"
    echo "[fail] fail_timeout" && exit 1
fi

# block
curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/block" >/dev/null &
sleep 1
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/block?input=1")
if contains "$OUT" "block"; then
    echo "[pass] block"
else
    echo "$OUT"
    echo "[fail] block" && exit 1
fi

# no_block
curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/no_block" >/dev/null &
sleep 1
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/no_block")
if contains "$OUT" "default test no_block"; then
    echo "[pass] no_block"
else
    echo "$OUT"
    echo "[fail] no_block" && exit 1
fi

# no_output
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/no_output")
if contains "$OUT" "done"; then
    echo "[pass] no_output"
else
    echo "$OUT"
    echo "[fail] no_output" && exit 1
fi

# input_validate
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/input_validate?input=123")
if contains "$OUT" "input_validate"; then
    echo "[pass] input_validate"
else
    echo "$OUT"
    echo "[fail] input_validate" && exit 1
fi

# retry_fail
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/retry_fail?input=123")
if contains "$OUT" "2 retries"; then
    echo "[pass] retry_fail"
else
    echo "$OUT"
    echo "[fail] retry_fail" && exit 1
fi

# json/parse
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/json/parse?input=%7B%22hello%22%3A%22world%22%7D")
if contains "$OUT" "hello"; then
    echo "[pass] json/parse"
else
    echo "$OUT"
    echo "[fail] json/parse" && exit 1
fi

# invalid
OUT=$(curl -sSk -H "$HEADER" "$URL/v1/pal/run/test/invalid")
if contains "$OUT" "invalid"; then
    echo "[pass] invalid"
else
    echo "$OUT"
    echo "[fail] invalid" && exit 1
fi

# html/index_cache
OUT=$(curl -sSk -H "$HEADER" -o /dev/null -w '%{content_type}' "$URL/v1/pal/run/html/index_cache")
if contains "$OUT" "text/html"; then
    echo "[pass] html/index_cache"
else
    echo "$OUT"
    echo "[fail] html/index_cache" && exit 1
fi

# DB PUT/GET
curl -sSk -XPUT -b "$COOKIE_FILE" -d 'UniqString123' "$URL/v1/pal/db/put?key=test" >/dev/null
OUT=$(curl -sSk -b "$COOKIE_FILE" "$URL/v1/pal/db/get?key=test")
if contains "$OUT" "UniqString123"; then
    echo "[pass] db/put/get"
else
    echo "$OUT"
    echo "[fail] db/put/get" && exit 1
fi

OUT=$(curl -sSk -b "$COOKIE_FILE" "$URL/v1/pal/db/get?key=key%20test/auth")
if contains "$OUT" "value"; then
    echo "[pass] db/register"
else
    echo "$OUT"
    echo "[fail] db/register" && exit 1
fi

# DB Delete
curl -sfk -XDELETE -b "$COOKIE_FILE" "$URL/v1/pal/db/delete?key=test" >/dev/null
OUT=$(curl -sSk -b "$COOKIE_FILE" "$URL/v1/pal/db/get?key=test")
if contains "$OUT" "value not found"; then
    echo "[pass] db/delete"
else
    echo "$OUT"
    echo "[fail] db/delete" && exit 1
fi

# GET Schedules
OUT=$(curl -sSk -b "$COOKIE_FILE" "$URL/v1/pal/schedules")
if contains "$OUT" "no_auth"; then
    echo "[pass] schedules/get"
else
    echo "$OUT"
    echo "[fail] schedules/get" && exit 1
fi

# GET Notifications
curl -sSk -b "$COOKIE_FILE" \
    -d '{"notification":"THE QUICK BROWN FOX JUMPED OVER THE LAZY DOGS BACK 1234567890","group":"json"}' \
    -H "content-type: application/json" -XPUT "$URL/v1/pal/notifications" >/dev/null

OUT=$(curl -sSk -b "$COOKIE_FILE" "$URL/v1/pal/notifications")
if contains "$OUT" "1234567890"; then
    echo "[pass] notifications/get"
else
    echo "$OUT"
    echo "[fail] notifications/get" && exit 1
fi

OUT=$(curl -sSk -b "$COOKIE_FILE" "$URL/v1/pal/notifications")
if contains "$OUT" "WEBHOOK"; then
    echo "[pass] notifications/webhook"
else
    echo "$OUT"
    echo "[fail] notifications/webhook" && exit 1
fi