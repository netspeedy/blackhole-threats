---
title: Signals and exit codes
description: The signals blackhole-threats handles, what each one does, and the codes it exits with.
---

## Signals

| Signal | What it does |
| --- | --- |
| `SIGUSR1` | Refreshes now: re-reads every feed and updates the routes, keeping the BGP sessions up. The next timed refresh counts from the end of this one. |
| `SIGHUP` | The same as `SIGUSR1`. |
| `SIGTERM` | Stops cleanly: logs `Shutdown requested`, stops the GoBGP server, which closes the sessions, and exits 0. |
| `SIGINT` | The same as `SIGTERM`. |

No signal reloads the YAML file or the flags; that takes a restart. `SIGHUP` is the usual reload
signal for a daemon, so blackhole-threats treats it as the nearest thing it has, a refresh, rather
than letting it end the process: a stray `SIGHUP` from a closing terminal or a tool that sends it
on rotation leaves the routes announced. It's logged with `signal=hangup`.

A refresh signal arriving during a refresh is acted on once that refresh finishes, so the daemon
refreshes again straight after. Several signals sent while it's busy are treated as one.

How to send them:

| Where it runs | Refresh | Stop |
| --- | --- | --- |
| Debian package | `sudo systemctl reload blackhole-threats` | `sudo systemctl stop blackhole-threats` |
| Container | `docker kill --signal=SIGUSR1 blackhole-threats` | `docker stop blackhole-threats` |
| By hand | `kill -USR1 <pid>` or `kill -HUP <pid>` | `kill <pid>`, or Ctrl-C |

## Exit codes

| Code | When |
| --- | --- |
| 0 | `-check-config` passed, `-version` or `-h` printed, `-once` finished, or the daemon stopped cleanly on `SIGINT` or `SIGTERM` |
| 1 | A fatal error: a bad config, an invalid feed, a bad `-log-format`, `-log-level` or `-refresh-rate`, GoBGP failing to start, or `Route sync failed` |
| 2 | An unknown flag or a flag value that doesn't parse |

`-once` exits 0 even when feeds fail, because a failed feed is logged and carried over rather than
treated as an error. A smoke test that needs every feed to work should check the output for
`Failed to read threat feed` as well as the exit code.

Under systemd, the unit's `Restart=on-failure` starts the daemon again 5 seconds after it exits
with an error.
