package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"math"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"gopkg.in/yaml.v3"
)

var ErrMalformedCommunity = errors.New("malformed community")
var ErrInvalidLocalASN = errors.New("invalid local ASN")
var ErrInvalidRouterID = errors.New("invalid router ID")
var ErrMissingFeedURL = errors.New("missing feed URL")
var ErrUnsupportedFeedScheme = errors.New("unsupported feed scheme")
var ErrInvalidNeighborPort = errors.New("invalid neighbor endpoint port")
var ErrConflictingNeighborPort = errors.New("conflicting neighbor endpoint port")
var ErrInvalidPrefixLimit = errors.New("invalid minimum prefix length")
var ErrNoDefaultCommunity = errors.New("no default community")

// The shortest prefixes announced when limits doesn't say otherwise. Nothing shorter than a /8 or
// a /16 belongs in a blackhole feed, and a feed line such as 0.0.0.0/0 or ::/0 would otherwise
// tell every router to drop everything.
const (
	DefaultMinIPv4PrefixLength = 8
	DefaultMinIPv6PrefixLength = 16
)

type Community uint32

func (c Community) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%d:%d", uint16(c>>16), uint16(c)), nil
}

func (c Community) String() string {
	s, _ := c.MarshalText()
	return string(s)
}

func (c *Community) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), ":")
	if len(parts) != 2 {
		return ErrMalformedCommunity
	}

	upper, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return fmt.Errorf("%w: upper %s", ErrMalformedCommunity, err)
	}
	lower, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return fmt.Errorf("%w: lower %s", ErrMalformedCommunity, err)
	}

	*c = Community(upper<<16 | lower)
	return nil
}

func (c Community) Uint32() uint32 {
	return uint32(c)
}

type Feed struct {
	URL       string    `yaml:"url"`
	Community Community `yaml:"community"`
	// AllowEmpty lets the feed give no prefixes without that counting as a failure, for a list
	// that's legitimately empty at times. Without it, a feed that gives no prefixes keeps its
	// community's previous routes.
	AllowEmpty bool `yaml:"allow_empty"`
}

func (f Feed) String() string {
	if f.Community == 0 {
		return f.URL
	}

	return fmt.Sprintf("%s[%s]", f.URL, f.Community)
}

type File struct {
	GoBGP  oc.BgpConfigSet `yaml:"gobgp"`
	Feeds  []Feed          `yaml:"feeds"`
	Limits Limits          `yaml:"limits"`
}

// Limits sets the shortest prefix, per address family, that blackhole-threats will announce. A
// prefix from a feed that's shorter is dropped rather than announced. The fields are pointers so
// that a key left out takes the default while an explicit 0 is rejected: there's deliberately no
// way to turn the check off.
type Limits struct {
	MinIPv4PrefixLength *int `yaml:"min_ipv4_prefix_length"`
	MinIPv6PrefixLength *int `yaml:"min_ipv6_prefix_length"`
}

// MinIPv4 returns the shortest IPv4 prefix length allowed.
func (l Limits) MinIPv4() int {
	if l.MinIPv4PrefixLength == nil {
		return DefaultMinIPv4PrefixLength
	}
	return *l.MinIPv4PrefixLength
}

// MinIPv6 returns the shortest IPv6 prefix length allowed.
func (l Limits) MinIPv6() int {
	if l.MinIPv6PrefixLength == nil {
		return DefaultMinIPv6PrefixLength
	}
	return *l.MinIPv6PrefixLength
}

// MinPrefixLength returns the shortest prefix length allowed for addr's address family.
func (l Limits) MinPrefixLength(addr netip.Addr) int {
	if addr.Is4() {
		return l.MinIPv4()
	}
	return l.MinIPv6()
}

// Validate checks each limit is a usable prefix length for its family. 0 is refused because it
// would let a default route through.
func (l Limits) Validate() error {
	if v := l.MinIPv4(); v < 1 || v > 32 {
		return fmt.Errorf("%w: limits.min_ipv4_prefix_length is %d, want 1 to 32", ErrInvalidPrefixLimit, v)
	}
	if v := l.MinIPv6(); v < 1 || v > 128 {
		return fmt.Errorf("%w: limits.min_ipv6_prefix_length is %d, want 1 to 128", ErrInvalidPrefixLimit, v)
	}
	return nil
}

type FeedList []Feed

var _ flag.Value = (*FeedList)(nil)

func (f *FeedList) String() string {
	values := make([]string, len(*f))
	for i := range len(*f) {
		values[i] = (*f)[i].String()
	}

	return "[" + strings.Join(values, " ") + "]"
}

func (f *FeedList) Set(value string) error {
	*f = append(*f, Feed{URL: value})
	return nil
}

func Load(path string) (File, error) {
	var cfg File

	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read config %q: %w", path, err)
	}

	normalized, err := normalizeNeighborEndpointPorts(data)
	if err != nil {
		return File{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(normalized))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return File{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	return cfg, nil
}

func (cfg File) Validate() error {
	if cfg.GoBGP.Global.Config.As == 0 {
		return ErrInvalidLocalASN
	}

	routerID, err := netip.ParseAddr(cfg.GoBGP.Global.Config.RouterId)
	if err != nil || !routerID.Is4() {
		return fmt.Errorf("%w: %q", ErrInvalidRouterID, cfg.GoBGP.Global.Config.RouterId)
	}

	if err := cfg.Limits.Validate(); err != nil {
		return err
	}

	return ValidateFeeds(cfg.Feeds, cfg.GoBGP.Global.Config.As)
}

// ValidateFeed checks a feed's URL. ValidateFeeds also checks that it has a community.
func ValidateFeed(feed Feed) error {
	source := strings.TrimSpace(feed.URL)
	if source == "" {
		return ErrMissingFeedURL
	}

	parsed, err := url.Parse(source)
	if err != nil {
		return fmt.Errorf("parse feed %q: %w", source, err)
	}

	switch parsed.Scheme {
	case "", "http", "https":
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedFeedScheme, parsed.Scheme)
	}
}

// ValidateFeeds checks every feed's URL, and that every feed will have a community when the local
// ASN is asn: one of its own, or the default if asn has one.
func ValidateFeeds(feeds []Feed, asn uint32) error {
	_, hasDefault := DefaultCommunity(asn)
	for i, feed := range feeds {
		if err := ValidateFeed(feed); err != nil {
			return fmt.Errorf("feed %d: %w", i+1, err)
		}
		if feed.Community == 0 && !hasDefault {
			return fmt.Errorf("feed %d: %w: %s has no community, and local AS %d is a 4-byte ASN, which doesn't fit in <ASN>:666; set community on the feed (a -feed flag can't)", i+1, ErrNoDefaultCommunity, feed.URL, asn)
		}
	}

	return nil
}

// DefaultCommunity returns <asn>:666, the community a feed carries when it doesn't set one. A
// standard community (RFC 1997) only has 16 bits for the ASN, so a 4-byte ASN has no default and
// ok is false. Truncating it would announce some other network's community: AS 4200000000 would
// come out as 59904:666.
func DefaultCommunity(asn uint32) (community Community, ok bool) {
	if asn > math.MaxUint16 {
		return 0, false
	}
	return Community(asn<<16 | 666), true
}
