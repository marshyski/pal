#!/bin/sh
set -eu

export PATH="$PATH:/pal"

PAL_CONFIG_DIR="${PAL_CONFIG_DIR:-/etc/pal}"
PAL_CONFIG_FILE="${PAL_CONFIG_FILE:-$PAL_CONFIG_DIR/pal.yml}"
PAL_ACTIONS_DIR="${PAL_ACTIONS_DIR:-$PAL_CONFIG_DIR/actions}"

cd /pal || { echo "error: cannot change into /pal directory" >&2; exit 1; }

# Restricted charset - safe in YAML scalars and shell without escaping
rand() { tr </dev/urandom -dc 'A-Za-z0-9_+=-' | head -c "$1"; }

if [ ! -f "$PAL_CONFIG_FILE" ]; then
    GLOBAL_TIMEZONE="${GLOBAL_TIMEZONE:-UTC}"
    GLOBAL_CONTAINER_CMD="${GLOBAL_CONTAINER_CMD:-docker}"
    GLOBAL_CMD_PREFIX="${GLOBAL_CMD_PREFIX:-/bin/sh -c}"
    GLOBAL_WORKDIR="${GLOBAL_WORKDIR:-/pal}"
    GLOBAL_DEBUG="${GLOBAL_DEBUG:-false}"

    HTTP_LISTEN="${HTTP_LISTEN:-0.0.0.0:8443}"
    HTTP_IPV6="${HTTP_IPV6:-false}"
    HTTP_TIMEOUT_MIN="${HTTP_TIMEOUT_MIN:-10}"
    HTTP_BODY_LIMIT="${HTTP_BODY_LIMIT:-90}"
    HTTP_REQ_PER_SEC="${HTTP_REQ_PER_SEC:-30}"
    HTTP_MAX_AGE="${HTTP_MAX_AGE:-3600}"
    HTTP_KEY="${HTTP_KEY:-$PAL_CONFIG_DIR/localhost.key}"
    HTTP_CERT="${HTTP_CERT:-$PAL_CONFIG_DIR/localhost.pem}"
    HTTP_SESSION_SECRET="${HTTP_SESSION_SECRET:-$(rand 32)}"
    HTTP_PROMETHEUS="${HTTP_PROMETHEUS:-false}"
    HTTP_DISABLE_UI="${HTTP_DISABLE_UI:-false}"
    HTTP_UPLOAD_DIR="${HTTP_UPLOAD_DIR:-/pal/upload}"
    # Pass raw YAML flow-style list, e.g. '[{header: X, value: Y}]'
    HTTP_HEADERS="${HTTP_HEADERS:-$(cat <<'EOF'

    - header: Access-Control-Allow-Origin
      value: "https://127.0.0.1:8443,https://localhost:8443"
EOF
)}"

    # users: either pass a full YAML flow list via HTTP_USERS, or let us
    # generate a single admin from ADMIN_USER/ADMIN_PASS.
    if [ -z "${HTTP_USERS:-}" ]; then
        ADMIN_USER="${ADMIN_USER:-pal}"
        ADMIN_PASS="${ADMIN_PASS:-$(rand 20)}"
        HTTP_USERS="[{user: \"${ADMIN_USER}\", pass: \"${ADMIN_PASS}\", role: admin}]"
        cat <<MSG
====================================================
 pal: generated admin credentials (first run only)
   user: ${ADMIN_USER}
   pass: ${ADMIN_PASS}
 Save these; they will not be shown again.
====================================================
MSG
    fi

    DB_ENCRYPT_KEY="${DB_ENCRYPT_KEY:-$(rand 32)}"
    DB_PATH="${DB_PATH:-$PAL_CONFIG_DIR/pal.db}"
    DB_IN_MEMORY="${DB_IN_MEMORY:-false}"

    NOTIFICATIONS_STORE_MAX="${NOTIFICATIONS_STORE_MAX:-100}"
    NOTIFICATIONS_WEBHOOKS="${NOTIFICATIONS_WEBHOOKS:-[]}"

    mkdir -p \
        "$PAL_CONFIG_DIR" \
        "$PAL_ACTIONS_DIR" \
        "$DB_PATH" \
        "$HTTP_UPLOAD_DIR"

    umask 077
    cat >"$PAL_CONFIG_FILE" <<EOF
global:
  timezone: "$GLOBAL_TIMEZONE"
  container_cmd: "$GLOBAL_CONTAINER_CMD"
  cmd_prefix: "$GLOBAL_CMD_PREFIX"
  working_dir: "$GLOBAL_WORKDIR"
  debug: $GLOBAL_DEBUG
http:
  listen: "$HTTP_LISTEN"
  ipv6: $HTTP_IPV6
  timeout_min: $HTTP_TIMEOUT_MIN
  body_limit: $HTTP_BODY_LIMIT
  req_per_sec: $HTTP_REQ_PER_SEC
  max_age: $HTTP_MAX_AGE
  key: "$HTTP_KEY"
  cert: "$HTTP_CERT"
  headers: $HTTP_HEADERS
  session_secret: "$HTTP_SESSION_SECRET"
  prometheus: $HTTP_PROMETHEUS
  disable_ui: $HTTP_DISABLE_UI
  upload_dir: "$HTTP_UPLOAD_DIR"
  users: $HTTP_USERS
db:
  encrypt_key: "$DB_ENCRYPT_KEY"
  path: "$DB_PATH"
  in_memory: $DB_IN_MEMORY
notifications:
  store_max: $NOTIFICATIONS_STORE_MAX
  webhooks: $NOTIFICATIONS_WEBHOOKS
EOF
    chmod -f 0400 "$PAL_CONFIG_FILE"
fi

exec /usr/bin/pal -c "$PAL_CONFIG_FILE" -d "$PAL_ACTIONS_DIR"
