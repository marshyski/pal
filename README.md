# pal

[goreport]: https://goreportcard.com/badge/github.com/marshyski/pal
[GoVer]: https://img.shields.io/github/go-mod/go-version/marshyski/pal?style=flat-square
[ci]: https://img.shields.io/github/actions/workflow/status/marshyski/pal/pal-ci.yml
[license]: https://img.shields.io/github/license/marshyski/pal
[tag]: https://img.shields.io/github/v/tag/marshyski/pal

![goreport]
![GoVer]
![ci]
![license]
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/marshyski/pal/badge)](https://scorecard.dev/viewer/?uri=github.com/marshyski/pal)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/9863/badge)](https://www.bestpractices.dev/projects/9863)
![OS](https://img.shields.io/badge/OS-linux-0078D4)
![CPU](https://img.shields.io/badge/CPU-x64%2C%20ARM64-0078D4)

**A simple API and UI for executing and scheduling system commands or scripts.** Great for webhooks and automating Linux server operations over HTTPS contained in a small binary.

## Table of Contents

- [Use Cases](#use-cases)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
  - [Local Development](#local-development)
  - [Docker](#docker)
  - [Vagrant](#vagrant)
  - [Systemd](#systemd)
  - [DEB & RPM Builds](#deb--rpm-builds)
- [YAML Definitions Configuration](#yaml-definitions-configuration)
- [API Endpoints](#api-endpoints)
  - [Command Execution](#command-execution)
  - [Key-Value Store](#key-value-store)
  - [Health Check](#health-check)
  - [File Management (Basic Auth)](#file-management-basic-auth)
  - [Notifications](#notifications)
  - [Crons](#crons)
  - [Actions](#actions)
- [Configurations](#configurations)
- [Built-In Variables](#built-in-variables)
  - [Env Variables](#env-variables)
  - [Notification Variables](#notification-variables)
- [YAML Server Configurations](#yaml-server-configurations)
- [Example Action Definition YML](#example-action-definition-yml)

## Use Cases

- Tiny CI / Job Server
- Remote Server Management
- Server Monitoring / Testing
- Homelab Automation

## Key Features

- Hide command output
- Cache last response / command output
- Create basic notifications inside pal
- Dynamic routing with easy YAML configurations
- Secure HTTP endpoints with auth header restriction
- File upload/download via a basic UI with Basic Auth
- Optional easy to use HTML UI (Works Offline/Air-Gap)
- Single binary (_20MB~_) with no external dependencies
- Control command execution: concurrent or sequential, background processes
- Secure key-value storage with BadgerDB (encrypted local filesystem database)
- Pass data to commands or scripts via env variables ([Built-In Env Variables](#env-variables))

## Quick Start

### Local Development

**Prerequisites:** Go 1.25 or higher

```bash
make
make certs
./pal -c ./pal.yml -d ./test
```

### Docker

#### Run default insecure test configs

```bash
make linux
make certs
# Choose between make debian / alpine
# Default insecure test configurations on debian:stable-slim
make debian
# Default insecure test configurations on alpine:latest
make alpine
```

#### Generate random secrets for one-time use

```bash
docker run -d --name=pal -p 8443:8443 -v "$(pwd)"/pal.yml:/etc/pal/pal.yml:ro -v "$(pwd)"/actions:/etc/pal/actions:ro --health-cmd 'curl -sfk https://127.0.0.1:8443/v1/pal/health || exit 1' --init --restart=unless-stopped pal:latest

# See generated random secrets
docker logs pal
```

#### Run with your own configs and mount db

```bash
mkdir -p ./actions ./upload ./pal.db
# If UID of docker user isn't UID/GID 101010 same as pal user inside container
sudo chown -Rf 101010:101010 ./

docker run -d --name=pal -p 8443:8443 -v "$(pwd)"/actions:/etc/pal/actions:ro -v "$(pwd)"/pal.yml:/etc/pal/pal.yml:ro -v "$(pwd)"/pal.db:/etc/pal/pal.db:rw -v "$(pwd)"/upload:/pal/upload:rw --health-cmd 'curl -sfk https://127.0.0.1:8443/v1/pal/health || exit 1' --init --restart=unless-stopped pal:latest
```

**Available Docker Run Env Variables:**

```bash
# Default insecure test values
-e HTTP_LISTEN="0.0.0.0:8443"
-e HTTP_TIMEOUT_MIN="10"
-e HTTP_BODY_LIMIT="90"
-e HTTP_MAX_AGE="3600"
-e HTTP_HEADERS='[]'
-e HTTP_PROMETHEUS='false'
-e HTTP_UI_UPLOAD_DIR='/pal/upload'
-e HTTP_UI_BASIC_AUTH='pal __Check_Container_Log_Output__'
-e HTTP_SESSION_SECRET='__Check_Container_Log_Output__'
-e DB_ENCRYPT_KEY='__Check_Container_Log_Output__'
-e DB_PATH='/etc/pal/pal.db'
-e GLOBAL_DEBUG='false'
-e GLOBAL_TIMEZONE='UTC'
-e GLOBAL_CMD_PREFIX='/bin/sh -c'
-e GLOBAL_WORKDIR='/pal'
-e NOTIFICATIONS_STORE_MAX='100'
```

### Vagrant

```bash
# Need nfpm to build RPMs / Debs
make install-deps
make vagrant # debian
make vagrant-rpm # rocky9
# If you want to ignore debs/rpm builds and installs just run:
# vagrant up
```

### Systemd

```bash
sudo cp pal.service /etc/systemd/system/pal.service
sudo systemctl daemon-reload
sudo systemctl start pal.service

# (optional) enable auto startup on system restarts
sudo systemctl enable pal.service
```

### DEB & RPM Builds

```bash
# Need nfpm to build RPM / DEB files
make install-deps
make pkg-all
```

**Default Access:** `https://127.0.0.1:8443` (See [Configurations](#configurations) to customize)

## YAML Definitions Configuration

```yaml
# REQUIRED Group name: e.g., /v1/pal/run/deploy
deploy:
  - # REQUIRED Action name: e.g., /v1/pal/run/deploy/app
    action: app
    # Description of action
    desc: Deploy app
    # Auth header: e.g., curl -H'X-Pal-Auth: secret_string_here'
    auth_header: X-Pal-Auth secret_string_here
    # Show command output (default: false)
    output: true
    # Set a default input if not set at request time
    input:
    # Run in background (default: false)
    background: false
    # Run concurrently (default: false)
    concurrent: true
    # Run in podman/docker container (default: null)
    container:
      # Container image to use
      image: alpine:latest
      # Run options
      options: --security-opt=no-new-privileges:true --cap-drop=ALL --net=none
    # Set action to run multiple cron style schedules
    crons:
      - "*****"
    # Set command timeout in seconds (default: 600 seconds/10 mins)
    timeout: 600
    # Set custom HTTP Response Headers
    headers:
      - header:
        value:
    # Validate input provided to run, valid options can be found here https://github.com/go-playground/validator?tab=readme-ov-file#baked-in-validations
    input_validate: required
    # Register / Put a key and value in the DB
    register:
      key: "$PAL_GROUP-$PAL_ACTION"
      value: "input=$PAL_INPUT status=$PAL_STATUS output=$PAL_OUTPUT"
      secret: false
    on_error:
      # Send notification when an error occurs using built-in vars $PAL_GROUP $PAL_ACTION $PAL_INPUT $PAL_OUTPUT
      notification: "deploy failed group=$PAL_GROUP action=$PAL_ACTION input=$PAL_INPUT status=$PAL_STATUS output=$PAL_OUTPUT"
      # Try cmd number of times
      retries: 1
      # Pause in seconds before running the next retry
      retry_interval: 10
      # Trigger webhook HTTP request with the name defined in the pal.yml file
      webhooks:
        - webhook_name_defined_in_pal.yml
      # Run action(s) on_error
      run:
        - group: group_name
          action: action_name
          # Input for run action on_error
          input: $PAL_OUTPUT
    on_success:
      # Send notification when no errors occurs using built-in vars $PAL_GROUP $PAL_ACTION $PAL_INPUT $PAL_OUTPUT
      notification: "deploy failed group=$PAL_GROUP action=$PAL_ACTION input=$PAL_INPUT status=$PAL_STATUS output=$PAL_OUTPUT"
      # Trigger webhook HTTP request with the name defined in the pal.yml file
      webhooks:
        - webhook_name_defined_in_pal.yml
      # Run action(s) when no errors occurs
      run:
        - group: group_name
          action: action_name
          # Input for run action when no errors occurs
          input: $PAL_OUTPUT
    # Command prefix can be anything e.g. python -c, pwsh -Command, etc. Default is /bin/sh -c
    cmd_prefix: /bin/sh -c
    # REQUIRED Command or script (use $PAL_INPUT for variables)
    cmd: echo "GROUP=$PAL_GROUP ACTION=$PAL_ACTION INPUT=$PAL_INPUT REQUEST=$PAL_REQUEST UPLOAD_DIR=$PAL_UPLOAD_DIR"
```

**Example Request**

```bash
curl -sk -H'X-Pal-Auth: secret_string_here' 'https://127.0.0.1:8443/v1/pal/run/deploy/app?input=helloworld2'

curl -sk -H'X-Pal-Auth: secret_string_here' -XPOST -d 'helloworld2' 'https://127.0.0.1:8443/v1/pal/run/deploy/app'
```

## API Endpoints

### Command Execution

Run command using either GET (query param) or POST (post body). Access last cached output of command run.

**Query Parameters:**

- `input`: input to the running script/cmd also known as parameter or argument
- `last_output`: return only the last ran output and do not trigger a run
- `last_success`: return only the last successful output and do not trigger a run
- `last_failure`: return only the last failure output and do not trigger a run

```js
GET                 /v1/pal/run/{{ group name }}/{{ action name }}?input={{ data }}
GET                 /v1/pal/run/{{ group name }}/{{ action name }}?last_output=true
POST {{ any data }} /v1/pal/run/{{ group name }}/{{ action name }}
```

- `group name` (**Required**): Key from your YAML config
- `action name` (**Required**): Action value associated with the group
- `data` (**Optional**): Data (text, JSON) passed to your command/script as `$PAL_INPUT`

### Key-Value Store

Get, put or dump all contents of the database. Meant to store small data <1028 characters in length (no limit, just recommendation).

```js
PUT {{ any data }} /v1/pal/db/put?key={{ key_name }}&secret={{ secret }}
GET                /v1/pal/db/get?key={{ key_name }}
GET                /v1/pal/db/dump
DELETE             /v1/pal/db/delete?key={{ key_name }}
```

- `any data` (**Required**): Any type of data to store
- `key name` (**Required**): Key to identify the stored data
- `secret` (**Optional**): Boolean true or false to hide value in UI
- `dump` returns all key-value pairs from DB in a JSON object

**cURL Key-Value Example**

```bash
curl -vsk -u 'username:password' -XPUT -d 'pal' 'https://127.0.0.1:8443/v1/pal/db/put?key=name&secret=true'
```

### Health Check

Basic healthcheck endpoint. Enable Prometheus configuration for metrics endpoint.

```
GET /v1/pal/health
```

- Returns "ok" response body

### File Management (Basic Auth)

Upload and download files using a web request when enabled in the configuration.

```js
GET  [BASIC AUTH] /v1/pal/ui/files (Browser HTML View)
GET  [BASIC AUTH] /v1/pal/ui/files/download/{{ filename }} (Download File)
POST [BASIC AUTH] /v1/pal/ui/files/upload (Multiform Upload)
```

- `filename` (**Required**): For downloading a specific file

**cURL Upload Example**

```bash
# Get Cookie
curl -sSk -XPOST -d 'username=pal' -d 'password=p@LLy5' --cookie-jar ./pal.cookie 'https://127.0.0.1:8443/v1/pal/ui/login'

# Use Cookie to Upload File
curl -sSk -XPOST -F files='@{{ filename }}' -b ./pal.cookie 'https://127.0.0.1:8443/v1/pal/ui/files/upload'
```

### Notifications

Create or get notifications and filter by group name.

```js
GET /v1/pal/notifications
GET /v1/pal/notifications?group={{ group_name }}
PUT {{ json_data }} /v1/pal/notifications
```

- `group_name` (**Optional**): Only show notifications for group provided

**cURL Notification Example**

```bash
curl -vks -u 'username:password' \
  -d '{"notification":"THE QUICK BROWN FOX JUMPED OVER THE LAZY DOGS BACK 1234567890","group":"json"}' \
  -H "content-type: application/json" -XPUT \
  'https://127.0.0.1:8443/v1/pal/notifications'
```

### Crons

Get configured cron actions or run cron action now.

```js
GET /v1/pal/crons
GET /v1/pal/crons?group={{ group }}&action={{ action }}&run={{ run }}
```

- `group` (**Optional**): group name
- `action` (**Optional**): action name
- `run` (**Optional**): keyword "now" is only supported at this time. Runs action now.

### Actions

Get actions configuration including last_output and other run stats.

```js
GET /v1/pal/action?group={{ group }}&action={{ action }}
GET /v1/pal/action?group={{ group }}&action={{ action }}&disabled={{ boolean }}
GET /v1/pal/actions
```

- `group` (**Required**): group name
- `action` (**Required**): action name
- `disabled` (**Optional**): disabled boolean

## Configurations

```yaml
Usage: pal [options] <args>
  -c,	Set configuration file path location, default is ./pal.yml
  -d,	Set action definitions file directory location, default is ./actions

Example: pal -c ./pal.yml -d ./actions
```

## Built-In Variables

### Env Variables

Every cmd run includes the below built-in env variables.

`PAL_UPLOAD_DIR` - Full directory path to upload directory

`PAL_GROUP` - Group name

`PAL_ACTION` - Action Name

`PAL_INPUT` - Input provided, override default value

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

### Notification Variables

When `OnError.Notification` or `OnSuccess.Notification` is configured for the action, you can use available substitution variables in the notification message:

`$PAL_GROUP` - Group name

`$PAL_ACTION` - Action name

`$PAL_INPUT` - Input provided, override default value

`$PAL_STATUS` - Status of action run

`$PAL_OUTPUT` - Command error output

## YAML Server Configurations

**See latest example reference, here:** [https://github.com/marshyski/pal/blob/main/pal.yml](https://github.com/marshyski/pal/blob/main/pal.yml)

## Example Action Definition YML

```yaml
monitor:
  - action: system
    desc: Get primary system stats for monitoring
    auth_header: X-Monitor-System q1w2e3r4t5
    output: true
    cmd: |
      echo '|===/ DOCKER STATS \===|'
      command -v docker 1>/dev/null && sudo docker stats --no-stream; echo

      echo '|===/ FREE MEMORY \===|'
      free -hbvtw; echo

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

All actions can be defined in one file, or split into multiple `.yml` files. The `-d` CLI argument tells the program what directory to verify action yml files.

**For more complete examples, see:** [https://github.com/marshyski/pal/tree/main/test](https://github.com/marshyski/pal/tree/main/test)
