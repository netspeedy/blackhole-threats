---
title: Feeds and communities
description: Where feeds come from, how they're grouped by community, what counts as a failure, which prefixes are too short to announce, how prefixes are summarised and what's kept when a feed fails.
---

A feed is a list of networks to blackhole, and a community is the label its routes carry. The
community is also the unit blackhole-threats refreshes in: feeds that share one are read,
summarised and kept or replaced together.

## Where feeds come from

A feed is either a local file or an `http://` or `https://` URL.

A value with no URL scheme is a file path. A relative path is opened from the daemon's working
directory, not from the config file's directory, and the packaged service and the container both
run from `/`, so give absolute paths there. `file://` URLs aren't accepted; neither is any scheme
other than `http` and `https`, and those stop the daemon at startup with
`unsupported feed scheme`.

HTTP feeds are fetched with a plain `GET` and a 30 second timeout, following redirects. Only a
`200 OK` counts; any other status fails the feed. There's no way to add headers or credentials, so
a feed that needs an API key has to carry it in the URL. Requests go through Go's standard
transport, which honours `HTTPS_PROXY`, `HTTP_PROXY` and `NO_PROXY` from the environment.

[Feed formats](/blackhole-threats/reference/feed-formats/) has the exact rules for reading text
and JSON.

## Grouping by community

Every feed carries a community, written `<as>:<value>`. A feed without one gets
`<local ASN>:666`, so with local AS 64520 that's `64520:666`. Feeds added with the `-feed` flag
can't set a community and always get the default.

On each refresh the feeds are grouped by community and the groups are handled one at a time, in
order of their community value. Within a group, every feed is read at the same time. Each group
ends with one of two lines: `Prepared community routes` if all of its feeds were read, or
`Keeping previous community routes after feed errors` if any of them failed.

## What counts as a failure

A feed fails when it can't be read: the file is missing or unreadable, the HTTP request errors or
times out, the status isn't 200, a text line is longer than 1 MiB, or a JSON feed isn't valid
JSON or holds something other than objects. Each failure logs `Failed to read threat feed` with
the feed and the error.

A feed that's read but has nothing usable in it isn't a failure. An empty file, an HTML error
page served with a 200, or a JSON object that wraps its list in another key reads as zero
prefixes, or close to it, and the group is rebuilt without them. If that's the group's only feed, every route it
had is withdrawn. Watch `prefixes_read` on `Parsed threat feed` for a source that suddenly drops
to 0.

## Short prefixes

A blackhole route for `0.0.0.0/0` would tell your routers to drop everything, so blackhole-threats
won't announce anything that broad. By default the shortest prefix it announces is a `/8` for
IPv4 and a `/16` for IPv6, and the config's
[`limits`](/blackhole-threats/reference/configuration/#limits) section changes either one.

A prefix shorter than its limit is dropped as soon as the feed is read. Each one logs
`Dropped prefix shorter than the minimum length` with the feed, the prefix and the limit, and the
count goes into `prefixes_dropped` on `Parsed threat feed` and `Prepared community routes`. The
feed itself isn't treated as failed: the rest of its prefixes are used as normal, because failing
it would keep its old routes announced for as long as that one line stayed in the feed.

Lists of unallocated address space, such as full bogon lists, describe it in big blocks, so
they're the feeds most likely to hit the limits. Expect a warning for each such line on every
refresh.

## Summarising

Each group's prefixes are summarised before they become routes. blackhole-threats masks every
prefix to its network boundary (`192.0.2.77/24` becomes `192.0.2.0/24`), drops duplicates and
prefixes that sit inside a broader one, and merges pairs of sibling prefixes into their parent,
repeating until nothing changes. Two `/25`s that make up a `/24` become the `/24`, and four
consecutive `/26`s end up as one `/24` too.

Merging stops at the limits. Two neighbouring IPv4 `/8`s stay as two `/8`s rather than becoming a
`/7`, which would be shorter than the default limit. The summarised list is checked against the
limits again before it becomes routes, as a backstop.

`Prepared community routes` logs the count before and after as `prefixes_read` and
`prefixes_summarized`. A big gap between them is normal for feeds that list many neighbouring
addresses.

Summarising only happens inside a group. Two groups that list the same prefix produce one route
carrying both communities, but a `/24` in one group and a `/25` inside it in another stay as two
routes, each with its own community.

## When a feed fails

If any feed in a group fails, the group keeps every route it had after the previous refresh,
logs `Keeping previous community routes after feed errors` with `carried_forward` set to how many
it kept, and the refresh carries on with the next group. Nothing from the group's other feeds is
used that time, even the ones that were read. The next refresh tries again from scratch.

The routes kept are only the ones from the previous refresh in this run. Right after a start or
restart there aren't any, so a group whose feed fails then announces nothing until a refresh
reads it. `-once` always starts from nothing, so a failure there never carries anything forward.

A group that keeps failing keeps announcing the same old routes, however stale they get. Treat
repeated carry-forward warnings for one community as something to fix, not to ignore.

## Choosing communities

Because a group fails as a whole, the community you give a feed decides what fails with it. Give
feeds their own community when they come from different providers, when you trust them
differently, when one is experimental, or when your routers should treat them differently. Share a
community only when the feeds are one policy bucket and you're happy for one failure to freeze the
lot.

The [example config](https://github.com/netspeedy/blackhole-threats/blob/main/examples/blackhole-threats.yaml)
gives almost every feed its own community; the two Team Cymru bogon lists share `64520:1101`
because they're one list split by address family.

Changing a feed's community is a config change, so it takes a restart, and the feed's routes
come back with the new community. Check that your router policy matches the new value before you
restart with it.
