package bgp

import (
	"context"
	"net/netip"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/golang/protobuf/ptypes/any"
	api "github.com/osrg/gobgp/v3/api"
	gobgpconfig "github.com/osrg/gobgp/v3/pkg/config"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/osrg/gobgp/v3/pkg/server"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/netspeedy/blackhole-threats/internal/config"
	"github.com/netspeedy/blackhole-threats/internal/feed"
	"github.com/netspeedy/blackhole-threats/internal/logging"
)

type Server struct {
	server   *server.BgpServer
	asn      uint32
	routerID string
	feeds    feedReader
}

type routeState struct {
	prefix      netip.Prefix
	communities []config.Community
}

type feedReader interface {
	ReadMany(ctx context.Context, feeds ...config.Feed) feed.ReadResult
}

// New starts the embedded GoBGP server with configSet. Feeds are read with limits applied, so
// nothing shorter than them is announced.
func New(configSet *oc.BgpConfigSet, limits config.Limits) (*Server, error) {
	bgpServer := server.NewBgpServer(
		server.LoggerOption(logging.NewGoBGPLogger(log.StandardLogger().GetLevel(), log.StandardLogger().Formatter)),
	)
	go bgpServer.Serve()

	if _, err := gobgpconfig.InitialConfig(context.Background(), bgpServer, configSet, true); err != nil {
		return nil, err
	}

	return &Server{
		server:   bgpServer,
		asn:      configSet.Global.Config.As,
		routerID: configSet.Global.Config.RouterId,
		feeds:    feed.Reader{Limits: limits},
	}, nil
}

func (s *Server) Run(ctx context.Context, feeds []config.Feed, refreshInterval time.Duration, refreshSignals <-chan os.Signal, runOnce bool) error {
	// main has already checked this, but a feed that ends up with no community must never reach
	// buildRouteTable, so check again rather than rely on it.
	if err := config.ValidateFeeds(feeds, s.asn); err != nil {
		return err
	}

	timer := time.NewTimer(0)
	defer timer.Stop()

	current := map[string]routeState{}
	for {
		select {
		case <-timer.C:
			started := time.Now()
			next := s.buildRouteTable(ctx, feeds, current)

			withdrawn, err := s.withdrawStale(current, next)
			if err != nil {
				return err
			}
			announced, err := s.announceFresh(current, next)
			if err != nil {
				return err
			}

			log.WithFields(log.Fields{
				"announced":     announced,
				"withdrawn":     withdrawn,
				"active_routes": len(next),
				"duration_ms":   time.Since(started).Milliseconds(),
			}).Info("Refresh completed")

			current = next
			if runOnce {
				return nil
			}
			timer.Reset(refreshInterval)
		case sig := <-refreshSignals:
			log.WithField("signal", sig.String()).Info("Refresh signal received")
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(0)
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Server) buildRouteTable(ctx context.Context, feeds []config.Feed, previous map[string]routeState) map[string]routeState {
	defaultCommunity, _ := config.DefaultCommunity(s.asn)
	grouped := map[config.Community][]config.Feed{}
	for _, source := range feeds {
		community := source.Community
		if community == 0 {
			// Run has refused feeds without a community when the local ASN has no default.
			community = defaultCommunity
		}
		grouped[community] = append(grouped[community], source)
	}

	communities := make([]config.Community, 0, len(grouped))
	for community := range grouped {
		communities = append(communities, community)
	}
	sort.Slice(communities, func(i, j int) bool { return communities[i] < communities[j] })

	routes := map[string]routeState{}
	for _, community := range communities {
		sources := grouped[community]
		result := s.feeds.ReadMany(ctx, sources...)

		if len(result.FailedSources) > 0 {
			carried := carryForwardCommunityRoutes(routes, previous, community)
			log.WithFields(log.Fields{
				"community":         community.String(),
				"feed_count":        len(sources),
				"failed_feed_count": len(result.FailedSources),
				"failed_feeds":      result.FailedSources,
				"carried_forward":   carried,
			}).Warn("Keeping previous community routes after feed errors")
			continue
		}

		log.WithFields(log.Fields{
			"community":           community.String(),
			"feed_count":          len(sources),
			"prefixes_read":       result.Count,
			"prefixes_dropped":    result.Dropped,
			"prefixes_summarized": len(result.Prefixes),
		}).Info("Prepared community routes")

		for _, prefix := range result.Prefixes {
			key := prefix.String()
			state := routes[key]
			state.prefix = prefix
			state.communities = append(state.communities, community)
			routes[key] = state
		}
	}

	for key, state := range routes {
		state.communities = uniqueCommunities(state.communities)
		routes[key] = state
	}

	return routes
}

func carryForwardCommunityRoutes(next, previous map[string]routeState, community config.Community) int {
	count := 0
	for key, state := range previous {
		if !slices.Contains(state.communities, community) {
			continue
		}

		current := next[key]
		current.prefix = state.prefix
		current.communities = append(current.communities, community)
		next[key] = current
		count++
	}

	return count
}

func (s *Server) withdrawStale(previous, next map[string]routeState) (int, error) {
	count := 0
	for key, route := range previous {
		if replacement, ok := next[key]; ok && slices.Equal(route.communities, replacement.communities) {
			continue
		}

		if err := s.publishRoute(route.prefix, nil); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

func (s *Server) announceFresh(previous, next map[string]routeState) (int, error) {
	count := 0
	for key, route := range next {
		if existing, ok := previous[key]; ok && slices.Equal(route.communities, existing.communities) {
			continue
		}

		if err := s.publishRoute(route.prefix, route.communities); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

func (s *Server) publishRoute(prefix netip.Prefix, communities []config.Community) error {
	nlri, err := anypb.New(&api.IPAddressPrefix{
		Prefix:    prefix.Addr().String(),
		PrefixLen: uint32(prefix.Bits()),
	})
	if err != nil {
		return err
	}

	originAttr, err := anypb.New(&api.OriginAttribute{Origin: 0})
	if err != nil {
		return err
	}

	attrs := []*any.Any{originAttr}
	var family *api.Family

	if prefix.Addr().Is4() {
		family = &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST}
		nextHopAttr, err := anypb.New(&api.NextHopAttribute{NextHop: s.routerID})
		if err != nil {
			return err
		}
		attrs = append(attrs, nextHopAttr)
	} else {
		family = &api.Family{Afi: api.Family_AFI_IP6, Safi: api.Family_SAFI_UNICAST}
		nextHopAttr, err := anypb.New(&api.MpReachNLRIAttribute{
			Family:   family,
			Nlris:    []*any.Any{nlri},
			NextHops: []string{"::ffff:" + s.routerID},
		})
		if err != nil {
			return err
		}
		attrs = append(attrs, nextHopAttr)
	}

	if len(communities) > 0 {
		communitiesAttr, err := anypb.New(&api.CommunitiesAttribute{
			Communities: communitiesToUint32(communities),
		})
		if err != nil {
			return err
		}
		attrs = append(attrs, communitiesAttr)
	}

	_, err = s.server.AddPath(context.Background(), &api.AddPathRequest{
		Path: &api.Path{
			Family:     family,
			Nlri:       nlri,
			Pattrs:     attrs,
			IsWithdraw: len(communities) == 0,
		},
	})

	return err
}

func communitiesToUint32(communities []config.Community) []uint32 {
	values := make([]uint32, len(communities))
	for i, community := range communities {
		values[i] = community.Uint32()
	}
	return values
}

func uniqueCommunities(communities []config.Community) []config.Community {
	if len(communities) < 2 {
		return communities
	}

	sort.Slice(communities, func(i, j int) bool { return communities[i] < communities[j] })
	result := communities[:1]
	for _, community := range communities[1:] {
		if community != result[len(result)-1] {
			result = append(result, community)
		}
	}
	return result
}

func (s *Server) Stop() error {
	return s.server.StopBgp(context.Background(), &api.StopBgpRequest{})
}
