package bgp

import (
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/netspeedy/blackhole-threats/internal/config"
	"github.com/netspeedy/blackhole-threats/internal/feed"
)

type stubFeedReader struct {
	results map[string]feed.ReadResult
}

func (s stubFeedReader) ReadMany(_ context.Context, feeds ...config.Feed) feed.ReadResult {
	if len(feeds) != 1 {
		return feed.ReadResult{}
	}
	return s.results[feeds[0].URL]
}

func TestBuildRouteTableKeepsPreviousCommunityOnFeedFailure(t *testing.T) {
	t.Parallel()

	defaultCommunity := config.DefaultCommunity(64512)
	otherCommunity := config.Community(64512<<16 | 777)

	server := &Server{
		asn: 64512,
		feeds: stubFeedReader{
			results: map[string]feed.ReadResult{
				"https://feeds.example.net/default.txt": {
					FailedSources: []string{"https://feeds.example.net/default.txt"},
				},
				"https://feeds.example.net/other.txt": {
					Prefixes: []netip.Prefix{
						netip.MustParsePrefix("203.0.113.0/24"),
					},
					Count: 1,
				},
			},
		},
	}

	previous := map[string]routeState{
		"198.51.100.0/24": {
			prefix:      netip.MustParsePrefix("198.51.100.0/24"),
			communities: []config.Community{defaultCommunity},
		},
	}

	next := server.buildRouteTable(context.Background(), []config.Feed{
		{URL: "https://feeds.example.net/default.txt"},
		{URL: "https://feeds.example.net/other.txt", Community: otherCommunity},
	}, previous)

	if len(next) != 2 {
		t.Fatalf("buildRouteTable() len = %d, want %d", len(next), 2)
	}

	carried, ok := next["198.51.100.0/24"]
	if !ok {
		t.Fatal("buildRouteTable() missing carried-forward prefix")
	}
	if !slices.Equal(carried.communities, []config.Community{defaultCommunity}) {
		t.Fatalf("carried communities = %v, want %v", carried.communities, []config.Community{defaultCommunity})
	}

	fresh, ok := next["203.0.113.0/24"]
	if !ok {
		t.Fatal("buildRouteTable() missing refreshed prefix")
	}
	if !slices.Equal(fresh.communities, []config.Community{otherCommunity}) {
		t.Fatalf("fresh communities = %v, want %v", fresh.communities, []config.Community{otherCommunity})
	}
}

func TestRunOnceReturnsAfterInitialRefresh(t *testing.T) {
	t.Parallel()

	server := &Server{
		asn:      64512,
		routerID: "192.0.2.1",
		feeds: stubFeedReader{
			results: map[string]feed.ReadResult{},
		},
	}

	if err := server.Run(context.Background(), nil, 2, nil, true); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

// An empty response used to read as a feed with nothing in it, and the community's routes were
// withdrawn. It now fails the feed, so the routes are carried forward like any other failure.
func TestBuildRouteTableCarriesForwardAnEmptyFeed(t *testing.T) {
	t.Parallel()

	community := config.Community(64512<<16 | 1200)
	source := filepath.Join(t.TempDir(), "feed.txt")
	if err := os.WriteFile(source, nil, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := &Server{asn: 64512, feeds: feed.Reader{}}
	previous := map[string]routeState{
		"198.51.100.0/24": {
			prefix:      netip.MustParsePrefix("198.51.100.0/24"),
			communities: []config.Community{community},
		},
	}

	next := server.buildRouteTable(context.Background(), []config.Feed{{URL: source, Community: community}}, previous)
	carried, ok := next["198.51.100.0/24"]
	if !ok || !slices.Equal(carried.communities, []config.Community{community}) {
		t.Fatalf("buildRouteTable() = %v, want 198.51.100.0/24 carried forward with %s", next, community)
	}
}

// Right after a start there's nothing to carry forward, so an empty feed announces nothing.
func TestBuildRouteTableAnnouncesNothingForAnEmptyFeedOnTheFirstRefresh(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "feed.txt")
	if err := os.WriteFile(source, nil, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := &Server{asn: 64512, feeds: feed.Reader{}}
	next := server.buildRouteTable(context.Background(), []config.Feed{{URL: source}}, map[string]routeState{})
	if len(next) != 0 {
		t.Fatalf("buildRouteTable() = %v, want no routes", next)
	}
}

// A feed that sets allow_empty is trusted to mean it: empty is a success, and its community's
// routes are withdrawn.
func TestBuildRouteTableWithdrawsAnEmptyFeedThatAllowsIt(t *testing.T) {
	t.Parallel()

	community := config.Community(64512<<16 | 1200)
	source := filepath.Join(t.TempDir(), "feed.txt")
	if err := os.WriteFile(source, nil, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := &Server{asn: 64512, feeds: feed.Reader{}}
	previous := map[string]routeState{
		"198.51.100.0/24": {
			prefix:      netip.MustParsePrefix("198.51.100.0/24"),
			communities: []config.Community{community},
		},
	}

	next := server.buildRouteTable(context.Background(), []config.Feed{{URL: source, Community: community, AllowEmpty: true}}, previous)
	if len(next) != 0 {
		t.Fatalf("buildRouteTable() = %v, want the community's routes withdrawn", next)
	}
}
