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

**Prerequisites:** Go 1.23 or higher

```bash
make
make certs
./pal -c ./pal.yml -d ./test/pal-defs.yml
```

**Default Access:** `https://127.0.0.1:8443` (See \*see [Configurations](#configurations) to customize)

## YAML Definitions Configuration

```yaml
# Resource name: e.g., /v1/pal/run/deploy
deploy:
  - # Target name: e.g., /v1/pal/run/deploy?target=app
    target: app
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
curl -sk -H'X-Pal-Auth: secret_string_here' 'https://127.0.0.1:8443/v1/pal/run/deploy?target=app&arg=helloworld2'

curl -sk -H'X-Pal-Auth: secret_string_here' -XPOST -d 'helloworld2' 'https://127.0.0.1:8443/v1/pal/run/deploy?target=app'
```

## API Endpoints

### Command Execution

```
GET             /v1/pal/run/{{ resource name }}?target={{ target name }}&arg={{ data }}
POST {{ data }} /v1/pal/run{{ resource name}}?target={{ target name }}
```

- `resource name` (**Required**): Key from your YAML config
- `target name` (**Required**): Target value associated with the resource
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

### Health Check

```
GET /v1/pal/health
```

### File Management (Basic Auth)

```
GET  [BASIC AUTH] /v1/pal/upload (HTML View)
GET  [BASIC AUTH] /v1/pal/upload/{{ filename }} (Download File)
POST [BASIC AUTH] /v1/pal/upload (Multiform Upload)
```

- `filename` (**Optional**): For downloading a specific file

**cURL Upload Example**

```bash
curl -vsk -F files='@{{ filename }}' -u 'X-Pal-Auth:PaLLy!@#890-' 'https://127.0.0.1:8443/v1/pal/upload'
```

## Configurations

```
Usage of pal:
  -c string
      Configuration file location (default "./pal.yml")
  -d string
      Definitions file location (default "./pal-defs.yml")
```

**Example Run**

```bash
./pal -c ./pal.yml -d ./pal-defs.yml
```

## YAML Server Configurations

**See latest example reference, here:** [https://github.com/perlogix/pal/blob/main/pal.yml](https://github.com/perlogix/pal/blob/main/pal.yml)

## Example `pal-defs.yml`

```yaml
# Get system stats
monitor:
  - target: system
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
curl -sk -H'X-Monitor-System: q1w2e3r4t5' 'https://127.0.0.1:8443/v1/pal/run/monitor?target=system'
```

**For a more complete example, see:** [https://github.com/perlogix/pal/blob/main/test/pal-defs.yml](https://github.com/perlogix/pal/blob/main/test/pal-defs.yml)
