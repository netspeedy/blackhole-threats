---
title: Configuration
description: Write the YAML file blackhole-threats runs from, the local BGP settings, the peers and the feeds, and check it before it goes live.
---

blackhole-threats reads one YAML file with two main sections, `gobgp`, the BGP settings, and
`feeds`, the threat feeds, plus an optional `limits` section. `-conf` names the file; without it the daemon reads
`blackhole-threats.yaml` in the working directory. The packaged service uses
`/etc/blackhole-threats.yaml` and the container `/config/blackhole-threats.yaml`.

Start from the [example config](https://github.com/netspeedy/blackhole-threats/blob/main/examples/blackhole-threats.yaml),
which the Debian package installs as `/etc/blackhole-threats.yaml`, with an unchanged copy among
the package's docs ([Files and paths](/blackhole-threats/reference/files/) has where). Its
addresses come from the documentation ranges and its feeds are public lists, so every value needs
checking before you use it.

## The local side

```yaml
gobgp:
  global:
    config:
      as: 64520
      routerid: "198.51.100.10"
```

`as` is your local ASN and can't be 0. `routerid` has to be an IPv4 address, even if every peer is
IPv6, because it's also the next hop on the routes the daemon announces. Quote it, so YAML reads
it as a string.

GoBGP listens on port 179 unless you add `port`. A different port is useful for a lab or a test
run without root:

```yaml
gobgp:
  global:
    config:
      as: 64520
      routerid: "198.51.100.10"
      port: 1179
```

## Peers

Each router that should receive the routes is a neighbor:

```yaml
gobgp:
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64520
    - config:
        neighboraddress: "2001:db8:10::1"
        peeras: 64520
```

A neighbor gets the routes of its own address family unless you list both under `afisafis`;
[BGP announcements](/blackhole-threats/how-it-works/bgp/#which-peers-get-which-routes) shows
how.

If a router listens on a port other than 179, give the neighbor a `port`:

```yaml
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64520
        port: 1179
```

That's a shorthand blackhole-threats adds: it becomes GoBGP's `transport.config.remoteport` for
that neighbor. You can set `transport.config.remoteport` directly instead, or spell it
`remote-port`, but if you set both and they disagree the config is rejected with
`conflicting neighbor endpoint port`.

:::caution[GoBGP's own key names won't work]
The `gobgp` block is GoBGP's configuration, but its keys are GoBGP's Go field names in lower case
with no hyphens: `routerid`, `neighboraddress`, `peeras`, `localaddresslist`. The hyphenated
names in GoBGP's documentation, such as `router-id` and `neighbor-address`, are unknown keys
here and stop the daemon with `field router-id not found in type oc.GlobalConfig`. The one
hyphenated spelling accepted is `remote-port`.
:::

Other GoBGP neighbor settings work the same way when you need them, such as `description`,
`authpassword` for TCP MD5, `transport.config.localaddress` to pick the source address, or
`ebgpmultihop`. [Config keys](/blackhole-threats/reference/configuration/#other-gobgp-settings)
lists the common ones.

## Feeds

```yaml
feeds:
  - url: https://www.spamhaus.org/drop/drop_v4.json
    community: 64520:1102
  - url: /var/lib/blackhole-threats/local-blocklist.txt
    community: 64520:1200
  - url: https://lists.blocklist.de/lists/all.txt
```

`url` is a file path or an `http://` or `https://` URL. Use absolute paths for files: a relative
one is opened from the daemon's working directory, which is `/` under systemd and in the
container. Under systemd, keep feed files out of `/home` and `/tmp`, because the unit's
`ProtectHome` and `PrivateTmp` settings hide them from the service.

`community` is written `<as>:<value>`, each part from 0 to 65535. Leave it out and the feed uses
`<local ASN>:666`, so the third feed above gets `64520:666`. Feeds that share a community succeed
or fail together; [Feeds and communities](/blackhole-threats/how-it-works/feeds/#choosing-communities)
covers how to choose.

:::caution[4-byte ASNs]
The default community puts the local ASN in its first half, which only has room for 16 bits. With
a local ASN above 65535 the default comes out wrong (AS 4200000000 gives `59904:666`), so give
every feed an explicit community.
:::

A feed that gives no prefixes, such as an empty response or an HTML error page served with a
200, counts as failed, so its community keeps the routes it had. If a feed can legitimately be
empty, such as a local list you clear when there's nothing to block, say so:

```yaml
feeds:
  - url: /var/lib/blackhole-threats/local-blocklist.txt
    community: 64520:1200
    allow_empty: true
```

Then an empty read succeeds and withdraws the feed's routes.

You can add feeds without editing the file with `-feed`, which you can repeat. Those always use
the default community and can't set `allow_empty`.

## Limits

blackhole-threats won't announce an IPv4 prefix shorter than `/8` or an IPv6 prefix shorter than
`/16`. If a feed lists one, such as `0.0.0.0/0`, it's dropped with a
`Dropped prefix shorter than the minimum length` warning and the rest of the feed is used as
normal. You only need a `limits` section to change those floors:

```yaml
limits:
  min_ipv4_prefix_length: 16
  min_ipv6_prefix_length: 32
```

The limits apply to every feed. Raising them is a cheap guard if none of your feeds should ever
list anything as broad as a `/8`. Lowering them is rarely what you want: a route that short
blackholes a large part of the internet. The lowest each one goes is 1, so a default route can
never be announced.

## Check it and roll it out

Check the file, then run a single refresh with it before the daemon uses it for real:

```bash
blackhole-threats -conf /etc/blackhole-threats.yaml -check-config
blackhole-threats -conf ./staging.yaml -once
```

`-check-config` catches YAML mistakes, unknown keys, a zero ASN, a router ID that isn't IPv4,
bad ports, bad communities, limits out of range and unsupported feed URLs. It doesn't start GoBGP, so GoBGP's own
objections to the `gobgp` block only show up when the daemon starts. `-once` does start it, and
reads every feed, so it shows whether the feeds are reachable and how many routes they make. Give
the copy you test with `-once` a `port` other than 179 if the live daemon is running on the same
host, or the two will fight over the port.

The daemon only reads the file at startup, so restart it to apply a change. `SIGUSR1` and
`systemctl reload` re-read the feeds, not the config.
