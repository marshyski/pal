#!/bin/bash

URL='https://127.0.0.1:8443/v1/pal/run'

# auth
OUT=$(curl -sk -H 'X-Pal-Auth: PaLLy!@#890-' "$URL/test/auth?input=helloworld")

if [ "$(echo "$OUT" | grep -c "helloworld")" = 1 ]; then
    echo "[pass] auth"
else
    echo "[fail] auth" && exit 1
fi

# unauth
OUT=$(curl -sk -H 'X-Pal-Auth: PaLLy!@#890-' "$URL/test/unauth")

if [ "$(echo "$OUT" | grep -c "helloworld")" = 1 ]; then
    echo "[pass] unauth"
else
    echo "[fail] unauth" && exit 1
fi

# emptycmd
OUT=$(curl -sk "$URL/test/emptycmd")

if [ "$(echo "$OUT" | grep -c "empty")" = 1 ]; then
    echo "[pass] emptycmd"
else
    echo "[fail] emptycmd" && exit 1
fi

# block
curl -sk $URL/test/block 1>/dev/null &
sleep 1
OUT=$(curl -sk "$URL/test/block")

if [ "$(echo "$OUT" | grep -c "ready")" = 1 ]; then
    echo "[pass] block"
else
    echo "[fail] block" && exit 1
fi

# noblock
curl -sk $URL/test/noblock 1>/dev/null &
sleep 1
OUT=$(curl -sk "$URL/test/noblock")

if [ "$(echo "$OUT" | grep -c "noblock")" = 1 ]; then
    echo "[pass] noblock"
else
    echo "[fail] noblock" && exit 1
fi

# norc
OUT=$(curl -sk "$URL/test/norc")

if [ "$(echo "$OUT" | grep -c "done")" = 1 ]; then
    echo "[pass] norc"
else
    echo "[fail] norc" && exit 1
fi

# json
OUT=$(curl -sk "$URL/json/newres?input=%7B%22hello%22%3A%22world%22%7D")
if [ "$(echo "$OUT" | grep -c "hello")" = 1 ]; then
    echo "[pass] json"
else
    echo "[fail] json" && exit 1
fi

# invalidaction
OUT=$(curl -sk "$URL/test2/invalidaction")
if [ "$(echo "$OUT" | grep -c "invalid")" = 1 ]; then
    echo "[pass] invalidaction"
else
    echo "[fail] invalidaction" && exit 1
fi

# emptyaction
OUT=$(curl -sk "$URL/test2/")
if [ "$(echo "$OUT" | grep -c "empty")" = 1 ]; then
    echo "[pass] emptyaction"
else
    echo "[fail] emptyaction" && exit 1
fi

# contenttype
OUT=$(curl -sk -o /dev/null -w '%{content_type}' "$URL/test2/contenttype")
if [ "$(echo "$OUT" | grep -c 'text/html')" = 1 ]; then
    echo "[pass] contenttype"
else
    echo "[fail] contenttype" && exit 1
fi
