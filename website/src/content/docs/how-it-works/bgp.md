---
title: BGP announcements
description: The embedded GoBGP server, which peers get which routes, what each route carries and when it's withdrawn.
---

blackhole-threats runs [GoBGP](https://github.com/osrg/gobgp) inside its own process. The
`gobgp` block of the config is handed to GoBGP as its configuration, and the daemon adds and
withdraws routes through GoBGP's API. There's no separate BGP daemon and no `gobgp` command-line
tool involved.

## Sessions

GoBGP listens for BGP connections on port 179 on all IPv4 and IPv6 addresses, and dials every
neighbor in the config. `gobgp.global.config.port` changes the listening port,
`gobgp.global.config.localaddresslist` limits the addresses it listens on, and a neighbor's
`config.port` changes the port it dials for that neighbor. Listening on 179 needs root or
`CAP_NET_BIND_SERVICE`; the packaged service and the container run as root.

Whether a session is iBGP or eBGP comes from the ASNs: a neighbor whose `peeras` equals the local
`as` is iBGP. The example config and the router policy examples on this site are iBGP, which is
the usual setup for an RTBH route server inside one network.

blackhole-threats does nothing with routes its peers send it. GoBGP may still accept them into
its own table, so if you don't want that, filter them on the router side.

## Which peers get which routes

A neighbor with no `afisafis` list gets the address family of its own address: an IPv4 neighbor
is sent the IPv4 routes and an IPv6 neighbor the IPv6 ones. That's why the example config lists
IPv4 and IPv6 neighbors separately: a router that should get both needs a session over each
family, or one session with both families listed.

To carry both families over one session, list them on the neighbor:

```yaml
neighbors:
  - config:
      neighboraddress: "198.51.100.1"
      peeras: 64520
    afisafis:
      - config:
          afisafiname: ipv4-unicast
      - config:
          afisafiname: ipv6-unicast
```

## What a route carries

Every route is a unicast prefix with:

- origin `IGP`
- for IPv4, the router ID as its next hop
- for IPv6, the router ID as an IPv4-mapped IPv6 next hop, such as `::ffff:198.51.100.10`
- a communities attribute with every community the prefix came from, sorted

Nothing else is set: no local preference, no MED, and no `NO_EXPORT`. The next hop is a
placeholder for your routers to replace, since a blackhole policy sets the route's action rather
than forwarding to it. Because the routes don't carry `NO_EXPORT`, make sure your routers don't
pass them on to other networks; [Router policy](/blackhole-threats/guides/router-policy/) has
examples that keep them in a dedicated filter chain.

## When routes change

Each refresh ends by comparing the new route table with the one it announced last time:

- a prefix with the same communities as before is left alone and not sent again
- a prefix that's gone is withdrawn
- a prefix whose communities changed is withdrawn and announced again with the new set
- a new prefix is announced

Withdrawals go first, then announcements. `Refresh completed` counts both, along with the number
of routes now in the table:

```text
time=2026-11-05T04:40:51Z level=info msg="Refresh completed" active_routes=3 announced=1 duration_ms=0 withdrawn=1
```

A route only changes when the feeds change, so a quiet refresh logs `announced=0 withdrawn=0`.

## When the daemon stops

On `SIGINT` or `SIGTERM` the daemon stops the GoBGP server, which closes every session. Your
routers then drop all of its routes, as they would for any peer that goes away. When it starts
again it announces everything from scratch after the first refresh. That makes a restart, an
upgrade or a container recreate a short gap in blackholing, and a config change can't be
applied without one.
