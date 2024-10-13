# pal

**A simple API and UI for executing and scheduling system commands or scripts.** Great for webhooks and automating Linux server operations over HTTPS contained in a small binary.

## Table of Contents

- [Use Cases](#use-cases)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
  - [Local Development](#local-development)
  - [Docker](#docker)
- [YAML Definitions Configuration](#yaml-definitions-configuration)
- [API Endpoints](#api-endpoints)
  - [Command Execution](#command-execution)
  - [Key-Value Store](#key-value-store)
  - [Health Check](#health-check)
  - [File Management (Basic Auth)](#file-management-basic-auth)
  - [Notifications](#notifications)
  - [Crons](#crons)
  - [Action](#action)
- [Configurations](#configurations)
- [Built-In Env Variables](#built-in-env-variables)
- [YAML Server Configurations](#yaml-server-configurations)
- [Example `pal-actions.yml`](#example-pal-actionsyml)

## Use Cases

- Homelab automation
- Simple job/CI server
- HTTP API for server management
- Sync small data in a simple secure Key/Value store

## Key Features

- Hide command output
- Cache last response / command output
- Create basic notifications inside pal
- Dynamic routing with easy YAML configurations
- Secure HTTP endpoints with auth header restriction
- File upload/download via a basic UI with Basic Auth
- Optional easy to use HTML UI (Works Offline/Air-Gap)
- Single binary (_<20MB_) with no external dependencies
- Control command execution: concurrent or sequential, background processes
- Secure key-value storage with BadgerDB (encrypted local filesystem database)
- Pass data to commands or scripts via env variables ([Built-In Env Variables](#built-in-env-variables))

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
-e HTTP_UI_BASIC_AUTH='admin p@LLy5'
-e DB_ENCRYPT_KEY='8c755319-fd2a-4a89-b0d9-ae7b8d26'
```

**Default Access:** `https://127.0.0.1:8443` (See [Configurations](#configurations) to customize)

## YAML Definitions Configuration

```yaml
# Group name: e.g., /v1/pal/run/deploy
deploy:
  - # Action name: e.g., /v1/pal/run/deploy/app
    action: app
    # Description of action
    desc: Deploy app
    # Auth header: e.g., curl -H'X-Pal-Auth: secret_string_here'
    auth_header: X-Pal-Auth secret_string_here
    # Show command output (default: false)
    output: true
    # Run in background (default: false)
    background: false
    # Run concurrently (default: false)
    concurrent: true
    # Set action to run to a cron style schedule
    cron: "*****"
    # Set command timeout in seconds (default: 600 seconds/10 mins)
    timeout: 600
    # Set custom HTTP Response Headers
    resp_headers:
      - header:
      - value:
    # Validate input provided to run, valid options can be found here https://github.com/go-playground/validator?tab=readme-ov-file#baked-in-validations
    input_validate: required
    on_error:
      # Send notification when an error occurs
      notification: "app deployment failed"
      # Try cmd number of times
      retries: 1
      # Pause in seconds before running the next retry
      retry_interval: 10
    # Command or script (use $PAL_INPUT for variables)
    tags:
      - deploy
    cmd: echo "helloworld" && echo "$PAL_INPUT"
```

### Example Request

```bash
curl -sk -H'X-Pal-Auth: secret_string_here' 'https://127.0.0.1:8443/v1/pal/run/deploy/app?input=helloworld2'

curl -sk -H'X-Pal-Auth: secret_string_here' -XPOST -d 'helloworld2' 'https://127.0.0.1:8443/v1/pal/run/deploy/app'
```

## API Endpoints

### Command Execution

**Query Parameters:**

- `input`: input to the running script/cmd also known as parameter or argument
- `last_output`: return only the last ran output and do not trigger a run, basically a cache

```
GET                 /v1/pal/run/{{ group name }}/{{ action name }}?input={{ data }}
GET                 /v1/pal/run/{{ group name }}/{{ action name }}?last_output=true
POST {{ any data }} /v1/pal/run/{{ group name }}/{{ action name }}
```

- `group name` (**Required**): Key from your YAML config
- `action name` (**Required**): Action value associated with the group
- `data` (**Optional**): Data (text, JSON) passed to your command/script as `$PAL_INPUT`

### Key-Value Store

```
PUT {{ any data }} /v1/pal/db/put?key={{ key_name }}
GET                /v1/pal/db/get?key={{ key_name }}
GET                /v1/pal/db/dump
DELETE             /v1/pal/db/delete?key={{ key_name }}
```

- `any data` (**Required**): Any type of data to store
- `key name` (**Required**): Key to identify the stored data
- `dump` returns all key value pairs from DB in a JSON object

**cURL Key-Value Example**

```bash
curl -vsk -H'X-Pal-Auth: PaLLy!@#890-' -XPUT -d 'pal' 'https://127.0.0.1:8443/v1/pal/db/put?key=name'

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
curl -vsk -F files='@{{ filename }}' -u 'admin:p@LLy5' 'https://127.0.0.1:8443/v1/pal/upload'
```

### Notifications

```
GET /v1/pal/notifications?group={{ group_name }}
PUT {{ json_data }} /v1/pal/notifications
```

- `group_name` (**Optional**): Only show notifications for group provided

**cURL Notification Example**

```bash
curl -vks -H'X-Pal-Auth: PaLLy!@#890-' \
  -d '{"notification":"THE QUICK BROWN FOX JUMPED OVER THE LAZY DOGS BACK 1234567890","group":"json"}' \
  -H "content-type: application/json" -XPUT \
  'https://127.0.0.1:8443/v1/pal/notifications'
```

### Crons

```
GET /v1/pal/crons
GET /v1/pal/crons?group={{ group }}&action={{ action }}&run={{ run }}
```

- `group` (**Required**): group name
- `action` (**Required**): action name
- `run` (**Required**): keyword "now" is only supported at this time. Runs action now.

### Action

```
GET /v1/pal/action?group={{ group }}&action={{ action }}
```

- `group` (**Required**): group name
- `action` (**Required**): action name

## Configurations

```
Usage: pal [options] <args>
  -a,	Set action definitions file path location, default is ./pal-actions.yml
  -c,	Set configuration file path location, default is ./pal.yml

Example: pal -a ./pal-actions.yml -c ./pal.yml
```

## Built-In Env Variables

`PAL_UPLOAD_DIR` - Full directory path to upload directory

`PAL_GROUP` - Group name

`PAL_ACTION` - Action Name

`PAL_INPUT` - Input provided

`PAL_REQUEST` - HTTP Request Context In JSON

```json
{
  "method": "",
  "url": "",
  "headers": { "": "" },
  "query_params": { "": "" },
  "body": ""
}
```

## YAML Server Configurations

**See latest example reference, here:** [https://github.com/marshyski/pal/blob/main/pal.yml](https://github.com/marshyski/pal/blob/main/pal.yml)

## Example `pal-actions.yml`

```yaml
monitor:
  - action: system
    desc: Get primary system stats for monitoring
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
curl -sk -H'X-Monitor-System: q1w2e3r4t5' 'https://127.0.0.1:8443/v1/pal/run/monitor/system'
```

**For a more complete example, see:** [https://github.com/marshyski/pal/blob/main/test/pal-actions.yml](https://github.com/marshyski/pal/blob/main/test/pal-actions.yml)
