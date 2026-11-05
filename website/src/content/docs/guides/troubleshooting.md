---
title: Troubleshooting
description: What to check when the config won't load, a feed keeps failing, routes don't appear or the route count looks wrong.
---

Start with the last lines the daemon logged and the most recent `Refresh completed`. They usually
say whether the problem is the config, a feed or the BGP side. The two safe commands to reach for
first:

```bash
blackhole-threats -conf /etc/blackhole-threats.yaml -check-config
blackhole-threats -conf /etc/blackhole-threats.yaml -once
```

Run `-once` with a config whose `port` isn't 179 if the service is running on the same host.

## The config won't load

The daemon exits with code 1 and one of these:

| Message | Usual cause |
| --- | --- |
| `Failed to load configuration` with `read config` | The file isn't at the `-conf` path, or the service can't read it |
| `Failed to load configuration` with `field ... not found in type ...` | An unknown or misspelt key. GoBGP keys are lower case without hyphens: `routerid`, not `router-id` |
| `Failed to load configuration` with `malformed community` | A community that isn't two numbers from 0 to 65535 separated by a colon |
| `Failed to load configuration` with `neighbor endpoint port` | A neighbor `port` that isn't 1 to 65535, or one that disagrees with `transport.config.remoteport` |
| `Invalid configuration` with `invalid local ASN` | `gobgp.global.config.as` is missing or 0 |
| `Invalid configuration` with `invalid router ID` | `routerid` is missing or isn't an IPv4 address |
| `Invalid configuration` or `Invalid feed configuration` with `missing feed URL` or `unsupported feed scheme` | A feed with an empty `url`, or a scheme other than `http` or `https` (in the file or a `-feed` flag) |
| `Refresh timer must be greater than zero` | `-refresh-rate` is 0 or negative |
| `Invalid log format` or `Invalid log level` | A value `-log-format` or `-log-level` doesn't take |

Under systemd the service restarts every 5 seconds after a failure, so the same line repeats in
the journal until the config is fixed. If a flag is the problem, look in `EXTRA_OPTS` in
`/etc/default/blackhole-threats`.

If `-check-config` passes but the daemon still stops at startup with a `component=gobgp` line or
`Failed to start BGP server`, GoBGP rejected part of the `gobgp` block, which `-check-config`
doesn't load. A common one is failing to listen: something else already has the port, the
process isn't root and the port is below 1024, or the host has no IPv6 and GoBGP can't listen on
`::` (set `localaddresslist: ["0.0.0.0"]` under `gobgp.global.config`).

## A feed keeps failing

`Failed to read threat feed` names the source and the error. Check that:

- a file still exists at that path, and that the service can see it (under systemd, not in
  `/home` or `/tmp`)
- the URL still answers with `200 OK` within 30 seconds from the host the daemon runs on
- a JSON feed is still valid JSON made of objects, and hasn't changed shape
- the provider hasn't started rate-limiting or blocking you

While it fails, its community keeps the routes from the last good refresh, logging
`Keeping previous community routes after feed errors`. That's safer than withdrawing them, but it
also means stale routes stay announced for as long as the feed stays broken.

## A feed reads zero prefixes

If `Parsed threat feed` shows `prefixes_read=0` and the feed isn't really empty, the response
probably isn't in a shape blackhole-threats reads. Common causes are an HTML page served with a
200, a JSON object that wraps the list in another key, or a text format with something attached
to each address, such as `ip=192.0.2.1`. [Feed formats](/blackhole-threats/reference/feed-formats/)
has the exact rules. A zero read counts as success, so the community's routes are withdrawn.

## A reload didn't apply my change

`SIGUSR1` and `systemctl reload` only re-read the feeds. A change to the YAML file, to
`/etc/default/blackhole-threats` or to container environment variables needs a restart.

## Routes don't appear on the router

- Check the BGP session is established on the router. If it isn't, check the ASNs, the addresses,
  any non-standard ports on both sides and that nothing blocks TCP between them.
- Check the router gets the right address family: an IPv4 session only carries IPv4 routes unless
  the neighbor lists `afisafis`.
- Check the router's import policy matches the communities in the config, and that it installs
  matches as blackholes.
- Check `Refresh completed` reports a non-zero `active_routes`, and that each community logged
  `Prepared community routes` with the counts you expect.
- `-once` isn't a test for this: it exits as soon as the refresh is done, usually before any
  session is up.

GoBGP's own lines, marked `component=gobgp`, show what it's doing with the peers, and
`-log-level debug` adds its debug output.

## The route count is lower than expected

- Summarising merged neighbouring prefixes or dropped ones inside broader prefixes. Compare
  `prefixes_read` with `prefixes_summarized`.
- A text feed puts several addresses on one line. Only the first prefix on each line is read.
- A feed changed upstream, or several feeds now overlap.
- A community is being carried forward after a failure, so it's still on older, smaller data.

## The container has no config

- The entrypoint only writes the placeholder to `/config/blackhole-threats.yaml`, and only when
  the `/config` mount is writable.
- If you set `BLACKHOLE_THREATS_CONF` to another path, the file has to exist there; the container
  logs `Configuration file not found` and exits otherwise.

`docker logs blackhole-threats` shows the entrypoint's lines before the daemon's, and
`docker exec blackhole-threats ls -l /config` shows what the container sees.
