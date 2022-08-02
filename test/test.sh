#!/bin/bash

URL='https://127.0.0.1:8443/v1/pal'

# auth
OUT=$(curl -sk -H 'X-Pal-Auth: pal-123' "$URL/test?target=auth&arg=helloworld")

if [ "$(echo "$OUT" | grep -c "helloworld")" = 2 ]; then
    echo "[pass] auth"
else
    echo "[fail] auth" && exit 1
fi

# unauth
OUT=$(curl -sk -H 'X-Pal-Auth: 123' "$URL/test?target=unauth")

if [ "$(echo "$OUT" | grep -c "unauth")" = 1 ]; then
    echo "[pass] unauth"
else
    echo "[fail] unauth" && exit 1
fi

# emptycmd
OUT=$(curl -sk "$URL/test?target=emptycmd")

if [ "$(echo "$OUT" | grep -c "empty")" = 1 ]; then
    echo "[pass] emptycmd"
else
    echo "[fail] emptycmd" && exit 1
fi

# block
curl -sk $URL/test?target=block 1>/dev/null &
sleep 1
OUT=$(curl -sk "$URL/test?target=block")

if [ "$(echo "$OUT" | grep -c "ready")" = 1 ]; then
    echo "[pass] block"
else
    echo "[fail] block" && exit 1
fi

# noblock
curl -sk $URL/test?target=noblock 1>/dev/null &
sleep 1
OUT=$(curl -sk "$URL/test?target=noblock")

if [ "$(echo "$OUT" | grep -c "noblock")" = 1 ]; then
    echo "[pass] noblock"
else
    echo "[fail] noblock" && exit 1
fi

# norc
OUT=$(curl -sk "$URL/test?target=norc")

if [ "$(echo "$OUT" | grep -c "done")" = 1 ]; then
    echo "[pass] norc"
else
    echo "[fail] norc" && exit 1
fi

# newres
OUT=$(curl -sk "$URL/test2?target=newres&arg=helloworld")
if [ "$(echo "$OUT" | grep -c "helloworld")" = 1 ]; then
    echo "[pass] newres"
else
    echo "[fail] newres" && exit 1
fi

# invalidtarget
OUT=$(curl -sk "$URL/test2?target=invalidtarget")
if [ "$(echo "$OUT" | grep -c "invalid")" = 1 ]; then
    echo "[pass] invalidtarget"
else
    echo "[fail] invalidtarget" && exit 1
fi

# emptytarget
OUT=$(curl -sk "$URL/test2?target=")
if [ "$(echo "$OUT" | grep -c "empty")" = 1 ]; then
    echo "[pass] emptytarget"
else
    echo "[fail] emptytarget" && exit 1
fi
