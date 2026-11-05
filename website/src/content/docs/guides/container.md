---
title: Container
description: Run the GHCR image with a mounted config directory, pass it flags, refresh it and restart it.
---

The image is `ghcr.io/netspeedy/blackhole-threats`, built for `linux/amd64` and `linux/arm64` on
Debian trixie with [s6-overlay](https://github.com/just-containers/s6-overlay). It reads its
config from `/config` and logs to standard output.

## Run it

```bash
docker run -d \
  -p 179:179 \
  -v "$PWD/config:/config" \
  --name blackhole-threats \
  ghcr.io/netspeedy/blackhole-threats:latest
```

`-p 179:179` lets your routers connect in; the daemon also dials them itself. With
`--network host` instead, it uses the host's addresses directly and `-p` isn't needed. The
daemon runs as root inside the container, so it can listen on 179.

`latest` follows stable releases. Pin a version tag such as `v1.2.3`, `1.2.3` or the minor line
`1.2` if you'd rather choose when to upgrade.

## The config directory

On start, the entrypoint looks for `/config/blackhole-threats.yaml`. If it isn't there, it writes
this placeholder and starts the daemon with it:

```yaml title="config/blackhole-threats.yaml"
# Default configuration
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.168.1.1"
  neighbors:
    - config:
        neighboraddress: "192.168.1.1"
        peeras: 64512

feeds:
#  - url: http://localhost/drop.txt
#    community: 64512:666
```

It has no feeds, so it announces nothing, but it does try to peer with `192.168.1.1`. Replace it
with your own config (the [configuration guide](/blackhole-threats/guides/configuration/) walks
through one) and restart the container. Local feed files can live in the same directory: refer to
them by their path inside the container, such as `/config/local-blocklist.txt`.

The mount has to be writable for that first start. If it's read-only, or you'd rather not get the
placeholder, put your config there before the first run.

## Environment

| Variable | Default | What it does |
| --- | --- | --- |
| `BLACKHOLE_THREATS_CONF` | `/config/blackhole-threats.yaml` | The config file to run with. If you point it anywhere else and the file isn't there, the container logs `Configuration file not found` and exits instead of writing a placeholder. |
| `BLACKHOLE_THREATS_EXTRA_OPTS` | empty | More flags, separated by spaces, added after `-conf`. |

For example, JSON logs, debug output and a 15 minute refresh:

```bash
docker run -d \
  -p 179:179 \
  -v "$PWD/config:/config" \
  -e BLACKHOLE_THREATS_EXTRA_OPTS="-log-format json -log-level debug -refresh-rate 15m" \
  --name blackhole-threats \
  ghcr.io/netspeedy/blackhole-threats:latest
```

`HTTPS_PROXY`, `HTTP_PROXY` and `NO_PROXY` pass straight through to the daemon if the feeds need a
proxy.

## Refresh, restart and logs

The daemon reloads nothing on its own, so restart the container after changing the config or the
environment:

```bash
docker restart blackhole-threats
```

A restart closes the BGP sessions, so your routers drop the routes until the daemon is back and
has refreshed. To read the feeds now without restarting, send the daemon `SIGUSR1`. In this image
the container's process 1 is s6-overlay's `/init` rather than the daemon, so signal the daemon
itself from inside the container:

```bash
docker exec blackhole-threats sh -c \
  'for p in /proc/[0-9]*; do [ "$(readlink "$p/exe")" = /usr/sbin/blackhole-threats ] && kill -USR1 "${p##*/}"; done'
```

The daemon logs `Refresh signal received` and refreshes. `docker stop` shuts the container down
through `/init`, which stops the daemon with `SIGTERM`; it logs `Shutdown requested` and
`Stopped blackhole-threats` on the way out.

```bash
docker logs -f blackhole-threats
```

shows the entrypoint's lines and then the daemon's. [Log messages](/blackhole-threats/reference/logs/)
lists them.
