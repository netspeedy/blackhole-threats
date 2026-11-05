---
title: Feed formats
description: How blackhole-threats decides a feed is text or JSON, and exactly what it reads from each.
---

Every feed is read as either plain text or JSON. Either way, what comes out is a list of IPv4 and
IPv6 prefixes. A bare address becomes a host route, `/32` for IPv4 and `/128` for IPv6, and
every prefix is masked to its network boundary, so `192.0.2.77/24` is read as `192.0.2.0/24`.
Values that aren't addresses or prefixes are skipped without an error.

A prefix shorter than the limits, `/8` for IPv4 and `/16` for IPv6 unless the config's `limits`
section says otherwise, is dropped with a warning after it's read, so a feed that lists
`0.0.0.0/0` or `::/0` never gets it announced. The rest of the feed is still used.

## Text or JSON

A feed is read as JSON when either of these is true:

- its path or URL ends in `.json`, `.jsonl` or `.ndjson` (in any case)
- it's an HTTP feed whose `Content-Type` is `application/json`, `application/jsonl`,
  `application/ndjson`, `application/x-ndjson`, `text/json` or anything ending in `+json`

Everything else is read as text. The extension check looks at the end of the whole URL, so
`https://example.com/drop.json?key=abc` doesn't count as `.json` and relies on the server's
`Content-Type` instead.

## Text feeds

Text is read line by line:

1. Leading and trailing whitespace is trimmed.
2. Blank lines, and lines starting with `#`, `;` or `//`, are skipped.
3. The line is split into words at spaces, tabs, commas and semicolons.
4. Quotes, brackets, braces, parentheses and angle brackets are trimmed from each end of each
   word.
5. The first word that's a prefix or an address is taken, and the rest of the line is ignored.

So each line gives at most one prefix. These all work:

```text
192.0.2.0/24
198.51.100.10
2001:db8:10::/48
"203.0.113.0/24"
[2001:db8:20::1]
1.10.16.0/20 ; SBL256894
198.51.100.20 # seen 2026-11-01
```

These don't:

```text
prefix=203.0.113.0/24
192.0.2.0/25, 192.0.2.128/25
```

The first has no separator between the name and the prefix, so the word never parses. The second
gives `192.0.2.0/25` and drops the rest of the line.

A line longer than 1 MiB fails the whole feed.

## JSON feeds

A JSON feed is either one array of objects, or a stream of objects one after another, which covers
JSONL and NDJSON. It's an array if the first character that isn't whitespace is `[`.

From each object, the first of these fields that holds a valid prefix or address is used:

1. `cidr`
2. `prefix`
3. `ip`
4. `address`

Other fields are ignored, and an object without any of them is skipped, which is how the metadata
line at the end of Spamhaus's DROP files is passed over:

```json
{"cidr":"203.0.113.0/24","rir":"TEST","sblid":"SBL1"}
{"ip":"198.51.100.10"}
{"address":"2001:db8:20::1"}
{"generated_at":"2026-04-19T00:00:00Z"}
```

The fields have to be strings. An entry that isn't an object (`["192.0.2.1"]`, for instance), a
field that's a number, or JSON that doesn't parse fails the whole feed. A single object that
wraps the list, such as `{"data": [...]}`, is read as one object with none of the four fields, so
it gives no prefixes and no error.

## Local files and HTTP

A local file is opened by path, relative to the working directory if it isn't absolute. Its
format is decided by its extension alone.

An HTTP feed is fetched with `GET`, a 30 second timeout for the whole request, Go's default
redirect handling (up to 10 redirects) and the proxy settings in `HTTPS_PROXY`, `HTTP_PROXY` and
`NO_PROXY`. Any status other than `200 OK` fails the feed. The request carries Go's default user
agent and no custom headers or credentials.

What a failure does to the feed's community is on
[Feeds and communities](/blackhole-threats/how-it-works/feeds/#when-a-feed-fails).
