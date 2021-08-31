# Deployer

This script can used to deploy docker images on your server.

Usage
---
1. Clone this repo: `git clone git@github.com:4lexvav/deployer.git`
1. Compile source code: `go build`
1. Define token to authorize requests: `export DEPLOYER_TOKEN=token`
1. Run executable: `./deployer`
1. Now you can start sending requests to deploy images:

```shell
curl --request POST 'localhost:3000/deploy/' \
    --header 'Authorization: token' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "container-name",
        "image": "docker.io/library/alpine:3.11",
        "token": "{your-docker-hub-token}",
        "user": "{your-docker-hub-username}",
        "network": "{network-name to connect container to}",
        "tty": true,
        "env": {
            "MY_VARIABLE": "VALUE"
        },
        "volumes": [
            {
                "source": "/home/centos/src",
                "target": "/var/www"
            }
        ]
    }'
```

#### Sending request above performs the following steps:
1. Pull image provided in `image` property of request.
1. Stop and kill existing container with name provided in `name` property.
1. Create new container based on given image applying all provided options.
1. Start new container.

Running script as systemd service:
---
1. Modify and copy `deployer.service` file to `/etc/systemd/system/` directory.
1. Optionally run `sudo systemctl edit deployer` command and add your environment variable to file, e.g.:

```
[Service]
Environment="DEPLOYER_TOKEN=token"
```

3. Start and enable service:

```shell
sudo systemctl start deployer
sudo systemctl enable deployer
```
4. Finally, you can check service status using these commands:

`systemctl status deployer` - shows the whole service status
`systemctl is-active deployer` - shows current service state

Sample GitHub Workflow to send request from GitHub Actions
---

```yaml
name: Deploy image to server

on:
  workflow_dispatch:
    inputs:
      IMAGE:
        description: Image with tag
        default: "docker.io/library/alpine:3.11"
        required: true
      CONTAINER:
        description: Container name
        default: alp
        required: true

env:
  HOST: "your-app.com:3000/deploy/"

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Send deploy request
        uses: wei/curl@v1
        with:
          args: |
            -f -v -X POST ${{ env.HOST }} \
            -H 'Authorization: ${{ secrets.DEPLOYER_TOKEN }}' \
            -H 'Content-Type: application/json' \
            -d '{\"name\":\"${{ github.event.inputs.CONTAINER }}\",\"image\":\"${{ github.event.inputs.IMAGE }}\",\"token\":\"${{ secrets.DOCKER_TOKEN }}\",\"user\":\"${{ secrets.DOCKER_USER }}\",\"network\":\"network-name\",\"tty\":true,\"env\":{\"ENV_VAR\":\"${{ secrets.ENV_VAR_VALUE }}\",\"ENV_VAR_2\":\"ENV_VAR2_VALUE\"},\"volumes\":[{\"source\":\"/home/centos/src\",\"target\":\"/var/www\"}]}'
```

Before use, make sure to define required secrets in your repository settings:
- `DEPLOYER_TOKEN`
- `DOCKER_TOKEN`
- `DOCKER_USER`