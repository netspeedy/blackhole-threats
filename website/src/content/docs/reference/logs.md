---
title: Log messages
description: The two log formats and every message blackhole-threats writes, with its level and fields.
---

blackhole-threats logs to standard output, one line per event. Under systemd that goes to the
journal, and in the container to `docker logs`.

## Formats

The default, `-log-format logfmt`, writes the time in UTC, the level and the message, then the
fields in alphabetical order:

```text
time=2026-11-05T04:38:15Z level=info msg="Prepared community routes" community=64520:1200 feed_count=1 prefixes_read=4 prefixes_summarized=3
```

A value is quoted, Go style, when it contains a space, a tab, a newline, a quote or an `=`. Levels
are `panic`, `fatal`, `error`, `warning`, `info`, `debug` and `trace`; note that warnings are
written `warning` even though the flag value is `warn`.

`-log-format json` writes one JSON object per line with the same fields, plus `time`, `level` and
`msg`:

```json
{"community":"64520:1200","feed_count":1,"level":"info","msg":"Prepared community routes","prefixes_read":4,"prefixes_summarized":3,"time":"2026-11-05T04:38:29Z"}
```

Lines from the embedded GoBGP server have `component=gobgp` and GoBGP's own messages and fields,
such as `msg="Add a peer configuration" component=gobgp key=198.51.100.1 topic=Peer`. Their
field names are lower-cased.

## Startup

| Message | Level | Fields |
| --- | --- | --- |
| `Starting blackhole-threats` | info | `tag_version`, `commit`, `build_date`, `go_version`, `go_os`, `go_arch`, `pid` |
| `Loaded runtime configuration` | info | `config_path`, `run_mode`, `refresh_interval`, `configured_feeds`, `cli_feeds`, `total_feeds`, `peer_count`, `local_as`, `router_id`, `default_community` |
| `Started blackhole-threats` | info | |

`tag_version` is the release tag the binary was built from, such as `v1.2.3` or `v1.2.3-rc.1`,
or `unknown` when the build's version isn't a tag. The Debian package's binary is stamped with
the version without its `v`, so it logs `tag_version=unknown`; `-version` shows the version it
has. `run_mode` is `daemon`, or `oneshot` with `-once`. `configured_feeds` counts the feeds in
the file, `cli_feeds` the ones from `-feed`, and `total_feeds` both.

## Each refresh

| Message | Level | Fields |
| --- | --- | --- |
| `Refresh signal received` | info | `signal` |
| `Parsed threat feed` | info | `source`, `prefixes_read` |
| `Failed to read threat feed` | error | `source`, `error` |
| `Prepared community routes` | info | `community`, `feed_count`, `prefixes_read`, `prefixes_summarized` |
| `Keeping previous community routes after feed errors` | warning | `community`, `feed_count`, `failed_feed_count`, `failed_feeds`, `carried_forward` |
| `Refresh completed` | info | `announced`, `withdrawn`, `active_routes`, `duration_ms` |

`Refresh signal received` comes before a refresh that `SIGUSR1` started, with
`signal="user defined signal 1"`. Each feed logs either `Parsed threat feed` or
`Failed to read threat feed`, and each community then logs either `Prepared community routes`
or `Keeping previous community routes after feed errors`.

- `prefixes_read` is the number of prefixes taken from a feed, or from all of a community's
  feeds, before summarising; `prefixes_summarized` is how many routes the community comes to after.
- `failed_feeds` lists the sources that failed, and `carried_forward` how many routes the community
  kept from the previous refresh.
- `announced` and `withdrawn` count the routes sent to GoBGP this refresh, `active_routes` the
  routes in the table afterwards, and `duration_ms` how long the refresh took.

## Stopping

| Message | Level | Fields |
| --- | --- | --- |
| `Shutdown requested` | info | `signal`, when it's known |
| `Completed single refresh cycle` | info | |
| `Stopped blackhole-threats` | info | |
| `Failed to stop BGP server cleanly` | warning | `error` |
| `Route sync failed` | error | `error` |
| `Failed to stop BGP server cleanly after runtime error` | warning | `error` |

`Shutdown requested` follows `SIGINT` or `SIGTERM`, with `signal=interrupt` or
`signal=terminated`. `-once` logs `Completed single refresh cycle` instead. `Route sync failed`
means GoBGP refused a route while the daemon was withdrawing or announcing; the daemon exits with
code 1 after it.

## Fatal errors

Each of these is the last line before the daemon exits with code 1.

| Message | Fields | Cause |
| --- | --- | --- |
| `Invalid log format` | `log_format`, `error` | `-log-format` isn't `logfmt` or `json` |
| `Invalid log level` | `log_level`, `error` | `-log-level` isn't a level |
| `Failed to load configuration` | `error` | The file can't be read or decoded, has an unknown key, a bad community or a bad neighbor port |
| `Invalid configuration` | `error` | The ASN is 0, the router ID isn't IPv4, or a feed in the file is invalid |
| `Invalid feed configuration` | `error` | A `-feed` value is invalid |
| `Refresh timer must be greater than zero` | `refresh_interval` | `-refresh-rate` is 0 or negative |
| `Failed to start BGP server` | `error` | GoBGP couldn't start with the `gobgp` block |

GoBGP can also stop the daemon with a fatal line of its own, marked `component=gobgp`, for
example when it can't listen on its port.

## Container entrypoint

The container's entrypoint logs these before the daemon starts, in the same logfmt shape but with
every field value quoted:

| Message | Level | Fields |
| --- | --- | --- |
| `Creating default configuration file` | info | `config_path` |
| `Created default configuration file` | info | `config_path` |
| `Configuration file not found` | error | `config_path` |

After `Configuration file not found` the container exits with code 1.
