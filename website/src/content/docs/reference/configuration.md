---
title: Config keys
description: Every key in the blackhole-threats YAML file, the GoBGP settings it passes through, and what makes the file fail to load.
---

The config is one YAML document with three top-level keys: `gobgp`, `feeds` and the optional
`limits`. It's decoded strictly: a key blackhole-threats or GoBGP doesn't know stops the daemon rather than being
ignored. The [configuration guide](/blackhole-threats/guides/configuration/) walks through
writing one.

## Required keys

| Key | Type | What it is |
| --- | --- | --- |
| `gobgp.global.config.as` | integer | The local ASN. Can't be 0. |
| `gobgp.global.config.routerid` | string | The BGP router ID, which must be an IPv4 address. It's also the next hop on announced routes. |

Everything else is optional, though without `gobgp.neighbors` nobody receives the routes and
without `feeds` there are none to send.

## gobgp.global.config

| Key | Type | Default | What it is |
| --- | --- | --- | --- |
| `as` | integer | none | The local ASN |
| `routerid` | string | none | The router ID, IPv4 |
| `port` | integer | `179` | The port GoBGP listens on. `-1` turns listening off, so GoBGP only dials out. |
| `localaddresslist` | list of strings | `["0.0.0.0", "::"]` | The addresses GoBGP listens on |

## gobgp.neighbors

A list, one entry per peer.

| Key | Type | Default | What it is |
| --- | --- | --- | --- |
| `config.neighboraddress` | string | none | The peer's address, IPv4 or IPv6 |
| `config.peeras` | integer | none | The peer's ASN. The same as `as` makes it iBGP. |
| `config.port` | integer | `179` | The port to dial on the peer, from 1 to 65535. A blackhole-threats shorthand for `transport.config.remoteport`. |
| `transport.config.remoteport` | integer | `179` | The same setting in GoBGP's own place. `remote-port` is accepted as another spelling. |
| `afisafis` | list | the neighbor's own family | The address families to send. Each entry is `config.afisafiname`, such as `ipv4-unicast` or `ipv6-unicast`. |

`config.port` and `transport.config.remoteport` can both be set as long as they agree; if they
don't, the file is rejected with `conflicting neighbor endpoint port`.

### Other GoBGP settings

The `gobgp` block is GoBGP's own `BgpConfigSet`, so any of its settings can go here. The key for
each is GoBGP's Go field name in lower case without separators, which isn't the hyphenated name
in GoBGP's documentation: `neighboraddress`, not `neighbor-address`. Some that come up with a
route server:

| Key under a neighbor | What it is |
| --- | --- |
| `config.description` | A free-text label for the peer |
| `config.authpassword` | A TCP MD5 password for the session (Linux only) |
| `config.localas` | A different local ASN for this peer |
| `config.admindown` | `true` to configure the peer without bringing the session up |
| `transport.config.localaddress` | The source address to dial from |
| `transport.config.passivemode` | `true` to wait for the peer to connect rather than dialling it |
| `ebgpmultihop.config.enabled`, `ebgpmultihop.config.multihopttl` | eBGP multihop |

GoBGP checks these itself when the daemon starts; `-check-config` only checks that the keys exist
and the values have the right types.

## feeds

A list, one entry per feed.

| Key | Type | Default | What it is |
| --- | --- | --- | --- |
| `url` | string | none, required | A file path, or an `http://` or `https://` URL. A relative path is read from the working directory. |
| `community` | string | `<local ASN>:666` | The BGP community the feed's routes carry, written `<as>:<value>` with each part from 0 to 65535. |

`community: 0:0` means the same as leaving it out. Feeds added with `-feed` have no community and
always use the default. For a local ASN above 65535 the default doesn't fit (only the ASN's low
16 bits are kept, so AS 4200000000 gives `59904:666`), so set `community` on every feed.

## limits

The shortest prefix blackhole-threats will announce, for each address family. Both keys are
optional, and a key you leave out keeps its default.

| Key | Type | Default | What it is |
| --- | --- | --- | --- |
| `min_ipv4_prefix_length` | integer | `8` | The shortest IPv4 prefix announced, from 1 to 32 |
| `min_ipv6_prefix_length` | integer | `16` | The shortest IPv6 prefix announced, from 1 to 128 |

```yaml
limits:
  min_ipv4_prefix_length: 8
  min_ipv6_prefix_length: 16
```

A prefix from a feed that's shorter than its family's limit is dropped with a warning and the
rest of the feed is used as normal, so a feed line of `0.0.0.0/0` or `::/0` is never announced.
Summarising never merges prefixes into one shorter than the limit either. There's no way to turn
the check off: 0 isn't accepted. [Feeds and
communities](/blackhole-threats/how-it-works/feeds/#short-prefixes) has the details.

## What stops the file loading

`-check-config` and startup both reject:

- a file that can't be read, or isn't valid YAML
- an unknown key anywhere, including a hyphenated GoBGP key
- `as` missing or 0
- `routerid` missing or not an IPv4 address
- a neighbor `config.port` that isn't a number from 1 to 65535, a `remoteport` that isn't 0 to
  65535, or the two disagreeing
- a feed with an empty `url`, or a scheme other than `http` or `https`, such as `ftp://` or
  `file://`
- a `community` that isn't two numbers from 0 to 65535 separated by one colon
- a `limits.min_ipv4_prefix_length` outside 1 to 32, or a `limits.min_ipv6_prefix_length`
  outside 1 to 128

The [troubleshooting guide](/blackhole-threats/guides/troubleshooting/#the-config-wont-load) has
the message each one logs.

## A complete example

The repository's [example config](https://github.com/netspeedy/blackhole-threats/blob/main/examples/blackhole-threats.yaml),
shortened:

```yaml
# Reference configuration using RFC 5737 / RFC 3849 documentation addresses.
# Replace the local ASN, router ID, neighbors, and feed set to match your
# network before deployment.
gobgp:
  global:
    config:
      as: 64520
      routerid: "198.51.100.10"
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64520
        # Optional: connect to this peer on a non-default TCP port.
        # Omit this line to use standard BGP port 179.
        # port: 1179
    - config:
        neighboraddress: "2001:db8:10::1"
        peeras: 64520

feeds:
  - url: http://localhost:41412/security/blocklist
    community: 64520:1100
  - url: https://team-cymru.org/Services/Bogons/fullbogons-ipv4.txt
    community: 64520:1101
  - url: https://team-cymru.org/Services/Bogons/fullbogons-ipv6.txt
    community: 64520:1101
  - url: https://www.spamhaus.org/drop/drop_v4.json
    community: 64520:1102
```

The full file has ten neighbors and eleven feeds.
