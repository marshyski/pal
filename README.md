# pal

**A simple API for executing system commands or scripts.** Ideal for automating Linux server operations over HTTPS.

## Use Cases

- Homelab automation
- Simple job/CI server
- HTTP API for server management
- Sync small non-critical data in Key/Value store

## Key Features

- Secure endpoints with auth header restrictions
- Pass variables to commands or scripts
- Control command execution: concurrent or sequential, background processes
- Show or hide command output
- Dynamic routing with YAML configurations
- Secure key-value storage with BadgerDB (encrypted local filesystem database)
- File upload/download via a basic UI with Basic Auth

## Quick Start

### Local Development

**Prerequisites:** Go 1.23 or higher

```bash
make
make certs
./pal -c ./pal.yml -d ./test/pal-actions.yml
```

### Docker

```bash
make linux
make certs
make docker # Default configurations
```

**Available Docker Run Env Variables:**

```bash
# Default values
-e HTTP_LISTEN="127.0.0.1:8443"
-e HTTP_TIMEOUT_MIN="10"
-e HTTP_BODY_LIMIT="12M"
-e HTTP_CORS_ALLOW_ORIGINS='["*"]'
-e HTTP_AUTH_HEADER='X-Pal-Auth PaLLy!@#890-'
-e HTTP_UI_BASIC_AUTH='X-Pal-Auth PaLLy!@#890-'
-e DB_ENCRYPT_KEY='8c755319-fd2a-4a89-b0d9-ae7b8d26'
```

**Default Access:** `https://127.0.0.1:8443` (See [Configurations](#configurations) to customize)

## YAML Definitions Configuration

```yaml
# Group name: e.g., /v1/pal/run/deploy
deploy:
  - # Action name: e.g., /v1/pal/run/deploy?action=app
    action: app
    # Auth header: e.g., curl -H'X-Pal-Auth: secret_string_here'
    auth_header: X-Pal-Auth secret_string_here
    # Show command output (default: false)
    output: true
    # Run in background (default: false)
    background: false
    # Run concurrently (default: false)
    concurrent: true
    # Set custom HTTP Response Headers
    response_headers:
      - header:
      - value:
    # Set custom HTTP Content-Type Response Header plain/text by default
    content_type: # application/json | text/html
    # Command or script (use $ARG for variables)
    cmd: echo "helloworld" && echo "$ARG"
```

## Example Request

```bash
curl -sk -H'X-Pal-Auth: secret_string_here' 'https://127.0.0.1:8443/v1/pal/run/deploy?action=app&arg=helloworld2'

curl -sk -H'X-Pal-Auth: secret_string_here' -XPOST -d 'helloworld2' 'https://127.0.0.1:8443/v1/pal/run/deploy?action=app'
```

## API Endpoints

### Command Execution

```
GET             /v1/pal/run/{{ group name }}?action={{ action name }}&arg={{ data }}
POST {{ data }} /v1/pal/run/{{ group name }}?action={{ action name }}
```

- `group name` (**Required**): Key from your YAML config
- `action name` (**Required**): Action value associated with the group
- `data` (**Optional**): Data (text, JSON) passed to your command/script as `$ARG`

### Key-Value Store

```
PUT {{ data }} /v1/pal/db/put?key={{ key_name }}
GET            /v1/pal/db/get?key={{ key_name }}
GET            /v1/pal/db/dump
DELETE         /v1/pal/db/delete?key={{ key_name }}
```

- `data` (**Required**): Data to store
- `key name` (**Required**): Key to identify the stored data
- `dump` returns all key value pairs from DB in a JSON object

**cURL Key-Value Example**

```bash
curl -vsk -H'x-pal-auth: PaLLy!@#890-' -XPUT -d 'pal' 'https://127.0.0.1:8443/v1/pal/db/put?key=name'

```

### Health Check

```
GET /v1/pal/health
```

- Returns "ok" response body

### File Management (Basic Auth)

```
GET  [BASIC AUTH] /v1/pal/ui/files (Browser HTML View)
GET  [BASIC AUTH] /v1/pal/ui/files/download/{{ filename }} (Download File)
POST [BASIC AUTH] /v1/pal/ui/files/upload (Multiform Upload)
```

- `filename` (**Required**): For downloading a specific file

**cURL Upload Example**

```bash
curl -vsk -F files='@{{ filename }}' -u 'X-Pal-Auth:PaLLy!@#890-' 'https://127.0.0.1:8443/v1/pal/upload'
```

### Notifications

```
GET /v1/pal/notifications?group={{ group_name }}
PUT {{ json_data }} /v1/pal/notifications
```

- `group_name` (**Optional**): Only show notifications for group provided

** cURL Notification Example**

```bash
curl -vks -H'X-Pal-Auth: PaLLy!@#890-' \
  -d '{"notification":"THE QUICK BROWN FOX JUMPED OVER THE LAZY DOGS BACK 1234567890","group":"json"}' \
  -H "content-type: application/json" -XPUT \
  'https://127.0.0.1:8443/v1/pal/notifications'
```

### Schedules

```
GET /v1/pal/schedules
GET /v1/pal/schedules?=name={{ name }}&run={{ run }}
```

- `name` (**Required**): group/action is name of scheduled action
- `run` (**Required**): keyword "now" is only supported at this time. Runs action now.

## Configurations

```
Usage of pal:
  -c string
      Configuration file location (default "./pal.yml")
  -a string
      Action definitions file location (default "./pal-actions.yml")
```

**Example Run**

```bash
./pal -c ./pal.yml -d ./pal-actions.yml
```

## YAML Server Configurations

**See latest example reference, here:** [https://github.com/marshyski/pal/blob/main/pal.yml](https://github.com/marshyski/pal/blob/main/pal.yml)

## Example `pal-actions.yml`

```yaml
# Get system stats
monitor:
  - action: system
    auth_header: X-Monitor-System q1w2e3r4t5
    concurrent: false
    background: false
    output: true
    cmd: |
      echo '|===/ DOCKER STATS \===|'
      command -v docker 1>/dev/null && sudo docker stats --no-stream; echo

      echo '|===/ FREE MEMORY \===|'
      free -g; echo

      echo '|===/ DISK SPACE \===|'
      df -hT; echo

      echo '|===/ TOP CPU \===|'
      ps -eo pid,ppid,cmd,%mem,%cpu --sort=-%cpu | head; echo

      echo '|===/ TOP MEMORY \===|'
      ps -eo pid,ppid,cmd,%mem,%cpu --sort=-%mem | head; echo

      echo '|===/ TOP OPEN FILES \===|'
      lsof 2>/dev/null | cut -d" " -f1 | sort | uniq -c | sort -r -n | head; echo

      echo '|===/ UPTIME AND LOAD \===|'
      uptime
```

**Example Request**

```bash
curl -sk -H'X-Monitor-System: q1w2e3r4t5' 'https://127.0.0.1:8443/v1/pal/run/monitor?action=system'
```

**For a more complete example, see:** [https://github.com/marshyski/pal/blob/main/test/pal-actions.yml](https://github.com/marshyski/pal/blob/main/test/pal-actions.yml)
