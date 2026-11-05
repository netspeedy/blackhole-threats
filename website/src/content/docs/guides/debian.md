---
title: Debian package
description: Run blackhole-threats as the packaged systemd service, change its options, refresh and restart it, and read its logs.
---

The Debian package from the [signed APT repository](/blackhole-threats/installation/#signed-apt-repository)
is the usual way to run blackhole-threats in production. It installs:

| Path | What it is |
| --- | --- |
| `/usr/sbin/blackhole-threats` | The daemon |
| `/etc/blackhole-threats.yaml` | The config, installed from the example |
| `/etc/default/blackhole-threats` | The config path and extra flags for the service |
| `blackhole-threats.service` | The systemd unit, in the system unit directory |
| `/usr/share/man/man8/blackhole-threats.8.gz` | The man page, `man 8 blackhole-threats` |
| `/usr/share/doc/blackhole-threats/examples/blackhole-threats.yaml` | A second copy of the example |

## After installing

The package doesn't change debhelper's defaults for the unit, so installing it enables
`blackhole-threats.service` and starts it straight away, with the example config. That config
peers with documentation addresses that don't exist, so nothing receives its routes, but it does
listen on port 179 and fetch the public feeds it lists.

Write your own config over `/etc/blackhole-threats.yaml` (the
[configuration guide](/blackhole-threats/guides/configuration/) walks through it), check it, and
restart the service to load it:

```bash
sudo editor /etc/blackhole-threats.yaml
sudo blackhole-threats -conf /etc/blackhole-threats.yaml -check-config
sudo systemctl restart blackhole-threats
sudo journalctl -u blackhole-threats -f
```

If you'd rather it didn't run until you're ready, stop and disable it after installing, then
`sudo systemctl enable --now blackhole-threats` once the config is right.

## Service options

The unit runs:

```text
/usr/sbin/blackhole-threats -conf $CONF $EXTRA_OPTS
```

with both values from `/etc/default/blackhole-threats`:

```bash title="/etc/default/blackhole-threats"
# Default settings for blackhole-threats.

CONF="/etc/blackhole-threats.yaml"

EXTRA_OPTS=""
```

Put any other flags in `EXTRA_OPTS`, separated by spaces, for example
`EXTRA_OPTS="-refresh-rate 30m -log-format json"`. Every flag is in the
[CLI reference](/blackhole-threats/reference/cli/). Restart the service after changing the file.
The same file is the place for `HTTPS_PROXY` if the feeds have to go through a proxy.

## Refresh or restart

```bash
sudo systemctl reload blackhole-threats    # read the feeds now
sudo systemctl restart blackhole-threats   # load a changed config or changed flags
```

`reload` sends the daemon `SIGUSR1`, which runs a refresh straight away: the feeds are read again
and the routes updated, and the BGP sessions stay up. It doesn't re-read the YAML file or
`/etc/default/blackhole-threats`.

`restart` stops the daemon, which closes its BGP sessions so your routers drop its routes, and
starts it again with the current files. The routes come back after the first refresh. Use it for
any config change.

## How the unit runs it

The daemon runs as root, so it can listen on port 179. The unit sets `NoNewPrivileges`,
`PrivateTmp`, `ProtectHome` and `ProtectSystem=full`. That means the service can't see files in
`/home` or the real `/tmp`, and `/usr`, `/boot` and `/etc` are read-only to it, so keep local feed
files somewhere like `/var/lib/blackhole-threats/` and give their full path.

If the daemon exits with an error, systemd restarts it after 5 seconds, and keeps doing so. A bad
config therefore shows up as the same `level=fatal` line in the journal every few seconds rather
than as a failed unit.

## Logs

The daemon writes to standard output, which the unit sends to the journal under the identifier
`blackhole-threats`:

```bash
sudo journalctl -u blackhole-threats -f
sudo journalctl -u blackhole-threats -n 100 --no-pager
```

Lines are logfmt, `time=... level=... msg="..."` followed by fields. Add `-log-format json` to
`EXTRA_OPTS` for one JSON object per line instead. [Log messages](/blackhole-threats/reference/logs/)
lists every message.

## Upgrades and removal

`apt upgrade` installs a new release and restarts the service. dpkg treats
`/etc/blackhole-threats.yaml` and `/etc/default/blackhole-threats` as configuration files, so your
edits survive upgrades, and if a new release changes the shipped copy dpkg asks which version to
keep. `apt remove` leaves them in place; `apt purge` removes them.
