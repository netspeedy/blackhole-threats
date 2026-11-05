---
title: Operations
description: A first deployment, rolling out changes, telling a refresh from a restart, what healthy logs look like and when to worry.
---

Day to day, blackhole-threats needs little attention: it refreshes on its own and only touches
routes that changed. Most of the work is rolling out config changes safely and noticing when a
feed goes bad.

## A first deployment

1. Copy the [example config](https://github.com/netspeedy/blackhole-threats/blob/main/examples/blackhole-threats.yaml)
   and replace the local ASN, the router ID, the neighbors and the feeds with your own.
2. Check it with `-check-config`.
3. Run it once with `-once` on a port other than 179, from a host that can reach the feeds, and
   check the `Loaded runtime configuration` line shows the `local_as`, `router_id`, `peer_count`
   and `default_community` you expect, and that every feed logs `Parsed threat feed`.
4. Put the [router policy](/blackhole-threats/guides/router-policy/) in place before the routes
   arrive.
5. Start the long-running service on 179 and watch for the first `Refresh completed`.
6. Check on a router that the sessions are up and the routes are installed as blackholes.

## Refresh or restart

There are two ways to make the daemon act now, and they do different things:

| | Refresh | Restart |
| --- | --- | --- |
| How | `SIGUSR1`, or `systemctl reload blackhole-threats` | `systemctl restart blackhole-threats`, `docker restart` |
| Reads the feeds again | Yes | Yes, on the first refresh |
| Reads the YAML file and flags again | No | Yes |
| BGP sessions | Stay up | Close and reconnect; routers drop the routes until the first refresh |
| Routes kept from failing feeds | Yes | No, the daemon starts with nothing to carry forward |

So refresh when a feed's contents changed and you don't want to wait for the timer, and restart
for any change to the config or the flags.

## Rolling out a config change

1. Edit a copy of the config.
2. Check it with `-check-config`.
3. If the change adds or removes feeds, changes communities or could change the route count a lot,
   run the copy with `-once` on another port and compare the `prefixes_read` and
   `prefixes_summarized` numbers with what the live daemon logs.
4. If communities changed, update the router policy first.
5. Put the copy in place and restart the service.
6. Watch for the first `Refresh completed` and check its `active_routes`.

Avoid restarting while a feed is failing if you rely on its routes being carried forward, since a
restart loses them.

## The refresh interval

The default is two hours (`-refresh-rate 2h`). Feeds that change often or matter a lot may warrant
something shorter, such as `30m`, but most public lists update a few times a day at most, and a
short interval mostly means more requests to the providers. `-refresh-rate` takes Go durations:
`90s`, `15m`, `1h30m`.

A shorter interval for a while is handy when you're investigating a feed. Set it in
`EXTRA_OPTS` or `BLACKHOLE_THREATS_EXTRA_OPTS`, restart, and put it back afterwards.

## What healthy looks like

At startup, `Loaded runtime configuration` with the values you expect. Then on every refresh, one
`Parsed threat feed` per feed, one `Prepared community routes` per community, and a
`Refresh completed`:

```text
time=2026-11-05T04:40:39Z level=info msg="Refresh completed" active_routes=3 announced=0 duration_ms=0 withdrawn=0
```

`announced` and `withdrawn` are usually small, since only changed routes are sent, and
`active_routes` moves gradually as the feeds change. `duration_ms` is the whole refresh, which is
mostly fetching: communities are read one after another, so it's roughly the slowest feed in each
community added together.

## When to look closer

- The same feed logs `Failed to read threat feed` on refresh after refresh. Its community is
  stuck on old routes until it's fixed.
- `Keeping previous community routes after feed errors` keeps appearing for one community.
- A feed's `prefixes_read` falls far below its usual number. A feed that gives no prefixes at all
  fails and keeps its old routes, but one that gives a handful where it used to give thousands is
  taken at its word.
- `active_routes` jumps or falls sharply after a feed or config change.
- `Dropped prefix shorter than the minimum length` from a feed that doesn't usually list such
  broad prefixes. Something upstream has probably gone wrong with it.
- No `Refresh completed` for longer than the refresh interval.
- The service keeps restarting with a `level=fatal` line: the config or a flag is wrong.

[Troubleshooting](/blackhole-threats/guides/troubleshooting/) has what to check for each.
