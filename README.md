# pal

A simple webhook API to run system commands or scripts. Great for triggering deployments or other Linux operational tasks.

## Features

- Auth header restriction
- Pass a variable to command or script
- Concurrent or one at a time command runs
- Run command in background
- Command output response or hidden response
- Dynamic routing based on YAML configurations
- Store Key Value pairs in local embedded BadgerDB
- Upload and download files
- Generate bcrypt password hashs

## Quick Start

Needs Go 1.23 or higher installed

```sh
make
make certs
./pal -c ./pal.yml -d ./test/test.yml
```

Pal runs as `https://127.0.0.1:8443` by default. To configure a different listening address and port see [Configurations](#configurations).


## YAML Spec

```yml
# string: resource name, /v1/pal/run/deploy
deploy:
  -
    # string: target name, /v1/pal/run/deploy?target=app
    target: app
    # string: header key and value, curl -H'X-Pal-Auth: some_pass_or_token'
    auth_header: X-Pal-Auth some_pass_or_token
    # bool: return command output, default false
    output: true
    # background: don't wait for response, default false
    background: false
    # bool: run concurrently or one at a time, default false
    concurrent: true
    # string: put command or call script, you can use $ARG
    cmd: echo "helloworld" && echo "$ARG"
```

# Example Request:

```sh
curl -sk -H'X-Pal-Auth: some_pass_or_token' 'https://127.0.0.1:8443/v1/pal/run/deploy?target=app&arg=helloworld2'
```

## Request Structure

```python
GET /v1/pal/run/{{ resource name }}?target={{ target name }}&arg={{ argument }}
```

- `resource name` (**Mandatory**): name of a YAML key
- `target name` (**Mandatory**): target value of a resource
- `argument` (**Optional**): argument to be passed with variable `ARG` to command or script

```python
PUT {{ data }} /v1/pal/db/put?key={{ key_name }}
GET            /v1/pal/db/get?key={{ key_name }}
DELETE         /v1/pal/db/delete?key={{ key_name }}
```

- `data` (**Mandatory**): whatever data you want to send
- `key name` (**Mandatory**): key namespace to store data

```python
GET /v1/pal/health
```

```python
GET  [BASIC AUTH] /v1/pal/upload (HTML View)
GET  [BASIC AUTH] /v1/pal/upload/{{ filename }} (Download File)
POST [BASIC AUTH] /v1/pal/upload (Multiform Upload)
```

- `filename` (**Optional**): Download file from API

cURL example to upload file
```sh
curl -vsk -F files='@{{ filename }}' -u 'X-Pal-Auth:PaLLy!@#890-' 'https://127.0.0.1:8443/v1/pal/upload'
```

```python
POST {{ data }} /v1/pal/bcrypt/gen
POST JSON {"password":"","hash":""} /v1/pal/bcrypt/compare
```

cURL example to compare
```sh
curl -skv -XPOST -d '{"password":"","hash":""}' -H' Content-Type: application/json' 'https://localhost:8443/v1/pal/bcrypt/compare'
```

## Configurations

```sh
Usage of pal:
  -c string
      Configuration file location (default "./pal.yml")
  -d string
      Definitions file location (default "./pal-defs.yml")
```

Example Run:

```sh
./pal -c ./pal.yml -d ./pal-defs.yml
```

## Example pal.yml

Create a monitor resource to get system stats. To see another example look at https://github.com/perlogix/pal/blob/main/test/test.yml

```yml
monitor:
  -
    target: system
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

Example Request:

```sh
curl -sk -H'X-Monitor-System: q1w2e3r4t5' 'https://127.0.0.1:8443/v1/pal/run/monitor?target=system'
```