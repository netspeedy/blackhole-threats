package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/osrg/gobgp/v3/pkg/config/oc"
)

func TestCommunityRoundTrip(t *testing.T) {
	t.Parallel()

	var community Community
	if err := community.UnmarshalText([]byte("64512:666")); err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}

	if got := community.String(); got != "64512:666" {
		t.Fatalf("String() = %q, want %q", got, "64512:666")
	}
}

func TestCommunityRejectsMalformedValue(t *testing.T) {
	t.Parallel()

	var community Community
	err := community.UnmarshalText([]byte("64512:not-a-number"))
	if !errors.Is(err, ErrMalformedCommunity) {
		t.Fatalf("UnmarshalText() error = %v, want ErrMalformedCommunity", err)
	}
}

func TestFileValidateAcceptsMinimalConfig(t *testing.T) {
	t.Parallel()

	cfg := File{
		GoBGP: oc.BgpConfigSet{
			Global: oc.Global{
				Config: oc.GlobalConfig{
					As:       64512,
					RouterId: "192.0.2.1",
				},
			},
		},
		Feeds: []Feed{
			{URL: "https://example.com/feed.txt"},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestFileValidateRejectsInvalidRouterID(t *testing.T) {
	t.Parallel()

	cfg := File{
		GoBGP: oc.BgpConfigSet{
			Global: oc.Global{
				Config: oc.GlobalConfig{
					As:       64512,
					RouterId: "not-an-ip",
				},
			},
		},
	}

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidRouterID) {
		t.Fatalf("Validate() error = %v, want ErrInvalidRouterID", err)
	}
}

func TestValidateFeedsRejectsMissingFeedURL(t *testing.T) {
	t.Parallel()

	err := ValidateFeeds([]Feed{{URL: ""}})
	if !errors.Is(err, ErrMissingFeedURL) {
		t.Fatalf("ValidateFeeds() error = %v, want ErrMissingFeedURL", err)
	}
}

func TestValidateFeedsRejectsUnsupportedFeedScheme(t *testing.T) {
	t.Parallel()

	err := ValidateFeeds([]Feed{{URL: "ftp://example.com/feed.txt"}})
	if !errors.Is(err, ErrUnsupportedFeedScheme) {
		t.Fatalf("ValidateFeeds() error = %v, want ErrUnsupportedFeedScheme", err)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
unexpected: true
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want unknown-key parse failure")
	}
}

func TestLoadRejectsUnknownNestedKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64512
        unexpected: true
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want nested unknown-key parse failure")
	}
}

func TestLoadMapsNeighborConfigPortToTransportRemotePort(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64512
        port: 1179
feeds:
  - url: https://example.com/feed.txt
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := cfg.GoBGP.Neighbors[0].Transport.Config.RemotePort; got != 1179 {
		t.Fatalf("neighbor remote port = %d, want %d", got, 1179)
	}
}

func TestLoadAcceptsHyphenatedNeighborRemotePort(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64512
      transport:
        config:
          remote-port: 1179
feeds:
  - url: https://example.com/feed.txt
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := cfg.GoBGP.Neighbors[0].Transport.Config.RemotePort; got != 1179 {
		t.Fatalf("neighbor remote port = %d, want %d", got, 1179)
	}
}

func TestLoadRejectsInvalidNeighborConfigPort(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64512
        port: 70000
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if !errors.Is(err, ErrInvalidNeighborPort) {
		t.Fatalf("Load() error = %v, want ErrInvalidNeighborPort", err)
	}
}

func TestLoadRejectsConflictingNeighborPorts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
  neighbors:
    - config:
        neighboraddress: "198.51.100.1"
        peeras: 64512
        port: 1179
      transport:
        config:
          remoteport: 1180
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if !errors.Is(err, ErrConflictingNeighborPort) {
		t.Fatalf("Load() error = %v, want ErrConflictingNeighborPort", err)
	}
}

func TestLimitsDefaultWhenLeftOut(t *testing.T) {
	t.Parallel()

	var limits Limits
	if limits.MinIPv4() != 8 || limits.MinIPv6() != 16 {
		t.Fatalf("default limits = /%d and /%d, want /8 and /16", limits.MinIPv4(), limits.MinIPv6())
	}
	if err := limits.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLoadReadsLimits(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
limits:
  min_ipv4_prefix_length: 24
  min_ipv6_prefix_length: 48
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Limits.MinIPv4() != 24 || cfg.Limits.MinIPv6() != 48 {
		t.Fatalf("limits = /%d and /%d, want /24 and /48", cfg.Limits.MinIPv4(), cfg.Limits.MinIPv6())
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLoadKeepsTheOtherDefaultWhenOneLimitIsSet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
limits:
  min_ipv6_prefix_length: 32
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Limits.MinIPv4() != 8 || cfg.Limits.MinIPv6() != 32 {
		t.Fatalf("limits = /%d and /%d, want /8 and /32", cfg.Limits.MinIPv4(), cfg.Limits.MinIPv6())
	}
}

func TestLoadRejectsUnknownLimitKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
limits:
  min_prefix_length: 8
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want unknown-key parse failure")
	}
}

func TestValidateRejectsOutOfRangeLimits(t *testing.T) {
	t.Parallel()

	zero, one, v4Max, v4Over, v6Max, v6Over := 0, 1, 32, 33, 128, 129
	tests := []struct {
		name   string
		limits Limits
		ok     bool
	}{
		{"IPv4 0", Limits{MinIPv4PrefixLength: &zero}, false},
		{"IPv4 33", Limits{MinIPv4PrefixLength: &v4Over}, false},
		{"IPv6 0", Limits{MinIPv6PrefixLength: &zero}, false},
		{"IPv6 129", Limits{MinIPv6PrefixLength: &v6Over}, false},
		{"IPv4 1 and IPv6 1", Limits{MinIPv4PrefixLength: &one, MinIPv6PrefixLength: &one}, true},
		{"IPv4 32 and IPv6 128", Limits{MinIPv4PrefixLength: &v4Max, MinIPv6PrefixLength: &v6Max}, true},
	}

	for _, tt := range tests {
		cfg := File{
			GoBGP: oc.BgpConfigSet{
				Global: oc.Global{Config: oc.GlobalConfig{As: 64512, RouterId: "192.0.2.1"}},
			},
			Limits: tt.limits,
		}
		err := cfg.Validate()
		if tt.ok && err != nil {
			t.Errorf("%s: Validate() error = %v, want nil", tt.name, err)
		}
		if !tt.ok && !errors.Is(err, ErrInvalidPrefixLimit) {
			t.Errorf("%s: Validate() error = %v, want ErrInvalidPrefixLimit", tt.name, err)
		}
	}
}

func TestLoadReadsAllowEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blackhole-threats.yaml")
	content := []byte(`
gobgp:
  global:
    config:
      as: 64512
      routerid: "192.0.2.1"
feeds:
  - url: /var/lib/blackhole-threats/local-blocklist.txt
    community: 64512:1200
    allow_empty: true
  - url: https://example.com/feed.txt
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Feeds[0].AllowEmpty || cfg.Feeds[1].AllowEmpty {
		t.Fatalf("allow_empty = %v and %v, want true and false", cfg.Feeds[0].AllowEmpty, cfg.Feeds[1].AllowEmpty)
	}
}
