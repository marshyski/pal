#!/bin/bash

URL='https://127.0.0.1:8443/v1/pal/run'

# auth
OUT=$(curl -sk -H 'X-Pal-Auth: PaLLy!@#890-' "$URL/test/auth?input=helloworld")

if [ "$(echo "$OUT" | grep -c "helloworld")" = 1 ]; then
    echo "[pass] auth"
else
    echo "[fail] auth" && exit 1
fi

# no_auth
OUT=$(curl -sk -H 'X-Pal-Auth: PaLLy!@#890-' "$URL/test/no_auth")

if [ "$(echo "$OUT" | grep -c "no_auth")" = 1 ]; then
    echo "[pass] no_auth"
else
    echo "[fail] no_auth" && exit 1
fi

# fail
OUT=$(curl -sk "$URL/test/fail")

if [ "$(echo "$OUT" | grep -c "exit")" = 1 ]; then
    echo "[pass] fail"
else
    echo "[fail] fail" && exit 1
fi

# block
curl -sk $URL/test/block 1>/dev/null &
sleep 1
OUT=$(curl -sk "$URL/test/block?input=1")

if [ "$(echo "$OUT" | grep -c "block")" = 1 ]; then
    echo "[pass] block"
else
    echo "[fail] block" && exit 1
fi

# no_block
curl -sk $URL/test/no_block 1>/dev/null &
sleep 1
OUT=$(curl -sk "$URL/test/no_block")

if [ "$(echo "$OUT" | grep -c "no_block")" = 1 ]; then
    echo "[pass] no_block"
else
    echo "[fail] no_block" && exit 1
fi

# no_output
OUT=$(curl -sk "$URL/test/no_output")

if [ "$(echo "$OUT" | grep -c "done")" = 1 ]; then
    echo "[pass] no_output"
else
    echo "[fail] no_output" && exit 1
fi

# input_validate
OUT=$(curl -sk "$URL/test/input_validate?input=123")

if [ "$(echo "$OUT" | grep -c "input_validate")" = 1 ]; then
    echo "[pass] input_validate"
else
    echo "[fail] input_validate" && exit 1
fi

# json/parse
OUT=$(curl -sk "$URL/json/parse?input=%7B%22hello%22%3A%22world%22%7D")
if [ "$(echo "$OUT" | grep -c "hello")" = 1 ]; then
    echo "[pass] json/parse"
else
    echo "[fail] json/parse" && exit 1
fi

# invalid
OUT=$(curl -sk "$URL/test/invalid")
if [ "$(echo "$OUT" | grep -c "invalid")" = 1 ]; then
    echo "[pass] invalid"
else
    echo "[fail] invalid" && exit 1
fi

# html/index_cache
OUT=$(curl -sk -o /dev/null -w '%{content_type}' "$URL/html/index_cache")
if [ "$(echo "$OUT" | grep -c 'text/html')" = 1 ]; then
    echo "[pass] html/index_cache"
else
    echo "[fail] html/index_cache" && exit 1
fi

# test/container_run
# OUT=$(curl -sk "$URL/test/container_run")
# if [ "$(echo "$OUT" | grep -c 'container_run')" = 1 ]; then
#     echo "[pass] test/container_run"
# else
#     echo "[fail] test/container_run" && exit 1
# fi
