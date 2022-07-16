# pal

A simple webhook API to run system commands or scripts. Great for triggering deployments or other Linux operational tasks.

## Features

- Auth header restriction
- Pass a variable to command or script
- Blocking and non-blocking command runs
- Command output response or hidden response
- Dynamic routing based on YAML configurations

## Quick Start

Needs Go 1.16 or higher installed

```sh
make
make certs
./pal -c ./test/test.yml
```

Pal runs as `https://127.0.0.1:8686` by default. To configure a different listening address and port see Configurations.


## YAML Spec

```yml
# string: resource name, /v1/pal/deploy
deploy:
  -
    # string: target name, /v1/pal/deploy?target=app
    target: app
    # string: header key and value, curl -H'X-Pal-Auth: some_pass_or_token'
    auth_header: X-Pal-Auth some_pass_or_token
    # bool: return command output, default false
    rc_output: true
    # bool: block to only one request at a time, default false 
    block: true
    # string: put command or call script, you can use $ARG
    cmd: echo "helloworld" && echo "$ARG"
```

# Example Request:

```sh
curl -sk -H'X-Pal-Auth: some_pass_or_token' 'https://127.0.0.1:8686/v1/pal/deploy?target=app&arg=helloworld2'
```

## Request Structure

```python
/v1/pal/{{ resource name }}?target={{ target name }}&arg={{ argument }}
```

- `resource name` (**Mandatory**): name of a YAML key
- `target name` (**Mandatory**): target value of a resource
- `argument` (**Optional**): argument to be passed with variable `ARG` to command or script


## Configurations

```sh
Usage of ./pal:
  -c string
    	Configuration file location (default "./pal.yml")
  -l string
    	Set listening address and port (default "127.0.0.1:8686")
```

Example Run:

```sh
./pal -c /dir/file.yml -l 0.0.0.0:8080
```

## Example pal.yml

Create a monitor resource to get system stats. To see another example look at https://github.com/perlogix/pal/blob/main/test/test.yml

```yml
monitor:
  -
    target: system
    auth_header: X-Monitor-System q1w2e3r4t5
    block: true
    rc_output: true
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
curl -sk -H'X-Monitor-System: q1w2e3r4t5' 'https://127.0.0.1:8686/v1/pal/monitor?target=system'
```