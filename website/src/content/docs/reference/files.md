---
title: Files and paths
description: Where the Debian package and the container keep the binary, the config and the logs, and the environment each reads.
---

blackhole-threats keeps no state on disk. It reads its config at startup and its feeds on every
refresh, and that's all it touches.

## Debian package

| Path | What it is |
| --- | --- |
| `/usr/sbin/blackhole-threats` | The daemon |
| `/etc/blackhole-threats.yaml` | The config the service runs with. Installed from the example, and kept across upgrades. |
| `/etc/default/blackhole-threats` | `CONF`, the config path, and `EXTRA_OPTS`, more flags for the service |
| `blackhole-threats.service` | The systemd unit, in the system unit directory |
| `/usr/share/man/man8/blackhole-threats.8.gz` | The man page |
| `/usr/share/doc/blackhole-threats/examples/blackhole-threats.yaml` | The example config, unchanged |
| `/usr/share/doc/blackhole-threats/` | The package's changelog and copyright |

The unit runs `/usr/sbin/blackhole-threats -conf $CONF $EXTRA_OPTS` as root, from `/`, with
`/etc/default/blackhole-threats` as its environment file. `ExecReload` sends `SIGUSR1`. It sets
`NoNewPrivileges`, `PrivateTmp`, `ProtectHome` and `ProtectSystem=full`, so feed files need to
live outside `/home` and `/tmp`, and it restarts the daemon 5 seconds after a failure. Logs go to
the journal under the identifier `blackhole-threats`.

## Container

| Path | What it is |
| --- | --- |
| `/usr/sbin/blackhole-threats` | The daemon |
| `/usr/local/bin/blackhole-threats-entrypoint` | The command the image runs, which writes the placeholder config if needed and starts the daemon with `-conf`, then `BLACKHOLE_THREATS_EXTRA_OPTS`, then any arguments after the image name |
| `/config` | Mount your config directory here |
| `/config/blackhole-threats.yaml` | The default config path |

| Variable | Default | What it is |
| --- | --- | --- |
| `BLACKHOLE_THREATS_CONF` | `/config/blackhole-threats.yaml` | The config path. Only the default path gets a placeholder if it's missing. |
| `BLACKHOLE_THREATS_EXTRA_OPTS` | empty | More flags, split on spaces |

The image's entrypoint is the entrypoint script, which execs the daemon, so the daemon is the
container's process 1 and receives `docker kill` and `docker stop` signals itself. It runs as root
from `/`, listens on port 179 (the image exposes it) and logs to standard output.

## Release archives and source builds

A release archive holds one file, the binary, named for its platform:
`blackhole-threats-linux-amd64`, `blackhole-threats-linux-arm64` or `blackhole-threats-linux-arm`.
`make build` writes `dist/blackhole-threats`. Neither comes with a config: without `-conf`, the
daemon looks for `blackhole-threats.yaml` in the directory you run it from.

## Environment

The daemon itself reads no variables of its own. The HTTP client honours `HTTPS_PROXY`,
`HTTP_PROXY` and `NO_PROXY` for feeds. Under systemd, set them in
`/etc/default/blackhole-threats`; in the container, pass them with `-e`.
