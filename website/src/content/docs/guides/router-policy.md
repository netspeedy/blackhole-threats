---
title: Router policy
description: Accept blackhole-threats' routes on your routers, match their communities and blackhole the traffic, with RouterOS examples.
---

blackhole-threats only announces routes. Whether traffic to those prefixes is dropped depends on
what your routers do with them, so each router that peers with it needs an import policy that:

- accepts routes from the blackhole-threats session and nothing else from it
- matches the communities you gave your feeds, such as `64520:666` or `64520:1101`
- turns a match into a blackhole (or unreachable, or discard) route instead of a normal one
- keeps the routes to itself, since they carry no `NO_EXPORT` community

The next hop on the routes is the daemon's router ID, or `::ffff:<router ID>` for IPv6. A
blackhole policy replaces it, so it never needs to be reachable.

It's also worth limiting how many prefixes the router accepts from the session, if your platform
supports it, so a feed that suddenly lists far more than usual can't flood the table. Note that
blackhole-threats doesn't filter short prefixes out of feeds; a feed that lists `0.0.0.0/0` gets
it announced. A router policy that rejects very short prefixes from this peer is cheap insurance.

## RouterOS v7

```text
/routing bgp template
set default as=64512 disabled=no routing-table=main
/routing bgp connection
add address-families=ip,ipv6 as=64512 disabled=no input.allow-as=2 .filter=threats-in local.role=ibgp \
    name=threats remote.address=192.168.1.2 routing-table=main templates=default
/routing filter rule
add chain=threats-in comment="Blackhole C&C and don't route or peer addresses" disabled=no rule=\
    "if (bgp-communities equal 64512:666) {set blackhole yes; accept}"
```

## RouterOS v6

```text
/routing bgp instance
set default as=64512
/routing bgp peer
add address-families=ip,ipv6 allow-as-in=2 in-filter=threats-in name=threats remote-address=\
    192.168.1.2 ttl=default
/routing filter
add action=accept address-family=ip bgp-communities=64512:666 chain=threats-in comment=\
    "Blackhole IPv4 C&C and don't route or peer addresses" protocol=bgp set-type=blackhole
add address-family=ipv6 bgp-communities=64512:666 chain=threats-in comment=\
    "Unreachable IPv6 C&C and don't route or peer addresses" protocol=bgp set-type=unreachable
```

Both examples assume AS 64512 on both sides, iBGP, blackhole-threats at `192.168.1.2` and the
default community `64512:666`. Change the ASN, the address and the communities to match your
config, and add a rule for each community you use. Then check what happens to a route that
matches no rule: it should be rejected, not installed as an ordinary route towards the daemon's
next hop. If your RouterOS version accepts unmatched routes, end the chain with an explicit
reject.

The v6 example blackholes IPv4 and marks IPv6 unreachable; the v7 one sets `blackhole` for both.
Either way, a peer only receives the address families it shares with blackhole-threats, so an IPv4
session carries IPv4 routes unless the neighbor's `afisafis` says otherwise
([BGP announcements](/blackhole-threats/how-it-works/bgp/#which-peers-get-which-routes)).

## Using several communities

Feeds with their own communities let routers treat them differently. You might blackhole the
bogon lists everywhere, blackhole a command-and-control list only on the edge routers, and only
log matches for a feed you're still evaluating. That's a matter of one filter rule per community
on each router, matching the values in your `feeds`.

If you change a feed's community, change the router policy first. Otherwise, after the restart,
its routes arrive with a community no rule matches and stop being blackholed.
