# Docker deployment

The primary documentation is the [repository README](../README.md); this file only
covers the two compose files in this directory.

Both of them **build this repository** rather than pulling an image. There is no
prebuilt image for this fork, and pulling `ofsm/ofsm` would give you upstream,
which has none of the features described in the root README.

## Files

| File | What it is |
|---|---|
| `docker-compose.simple.yaml` | The manager alone, published on port 80. Use this behind your own reverse proxy, or on a machine you reach over VPN. |
| `docker-compose.yaml` | The manager behind Traefik, with automatic Let's Encrypt certificates for `DOMAIN_NAME`. Publishes 80/443 and the game's `34197/udp`. |
| `.env` | Template for the variables the composes read: `RCON_PASS`, and for the Traefik variant `DOMAIN_NAME` and `EMAIL_ADDRESS`. |

## Running it

```sh
export ADMIN_USER=admin
export ADMIN_PASS='a password you will remember'
export RCON_PASS='a password the game and the manager share'
export COOKIE_ENCRYPTION_KEY="$(openssl rand -base64 32)"

docker compose -f docker/docker-compose.simple.yaml up -d --build
```

For the Traefik variant, also set `DOMAIN_NAME` and `EMAIL_ADDRESS` (either in
`.env` or in your shell), then:

```sh
docker compose -f docker/docker-compose.yaml up -d --build
```

Every variable in the `environment:` section is required, and the container
refuses to start without the four manager ones: there is no generated random
password and no default to fall back on. The full list, including the optional
`FACTORIO_VERSION`, `COOKIE_SECURE`, `FSM_AUTOSTART` and `FSM_RCON_PORT`, is in
the root README's **Configuration** section.

The first start downloads the Factorio headless server, so the web UI appears a
minute or so after `up -d` reports success. Both composes carry a healthcheck that
accounts for that download, and cap the container's log at 10 MB × 3 files.

State lives in `./fsm-data` and, for the simple layout, `./factorio-data`. Backing
those two directories up is backing up the manager.

## Development

`build.sh` in this directory deletes old executables and `node_modules` (it runs
`make build`) and tags the image `factorio-server-manager:dev`.

### Creating release bundles

`Dockerfile-build` produces the release bundles with BuildKit. From the repository
root:

```sh
DOCKER_BUILDKIT=1 docker build --no-cache -f docker/Dockerfile-build -t ofsm-build --target=build -o dist .
```
