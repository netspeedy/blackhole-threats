package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/netspeedy/blackhole-threats/internal/bgp"
	"github.com/netspeedy/blackhole-threats/internal/buildinfo"
	"github.com/netspeedy/blackhole-threats/internal/config"
	"github.com/netspeedy/blackhole-threats/internal/logging"
)

func runMode(runOnce bool) string {
	if runOnce {
		return "oneshot"
	}

	return "daemon"
}

func logStartup(cfgPath string, cfg config.File, configuredFeeds, extraFeeds int, refreshInterval time.Duration, runOnce bool) {
	log.WithFields(log.Fields{
		"tag_version": buildinfo.TagVersion(),
		"commit":      buildinfo.Commit,
		"build_date":  buildinfo.BuildDate,
		"go_version":  runtime.Version(),
		"go_os":       runtime.GOOS,
		"go_arch":     runtime.GOARCH,
		"pid":         os.Getpid(),
	}).Info("Starting blackhole-threats")

	log.WithFields(log.Fields{
		"config_path":            cfgPath,
		"run_mode":               runMode(runOnce),
		"refresh_interval":       refreshInterval.String(),
		"configured_feeds":       configuredFeeds,
		"cli_feeds":              extraFeeds,
		"total_feeds":            configuredFeeds + extraFeeds,
		"peer_count":             len(cfg.GoBGP.Neighbors),
		"local_as":               cfg.GoBGP.Global.Config.As,
		"router_id":              cfg.GoBGP.Global.Config.RouterId,
		"default_community":      defaultCommunityField(cfg.GoBGP.Global.Config.As),
		"min_ipv4_prefix_length": cfg.Limits.MinIPv4(),
		"min_ipv6_prefix_length": cfg.Limits.MinIPv6(),
	}).Info("Loaded runtime configuration")
}

// defaultCommunityField is the default community for the startup log, or "none" for a 4-byte ASN,
// which doesn't have one.
func defaultCommunityField(asn uint32) string {
	community, ok := config.DefaultCommunity(asn)
	if !ok {
		return "none"
	}
	return community.String()
}

// cliFlags holds the command-line flags' values.
type cliFlags struct {
	cfgPath     string
	checkConfig bool
	debug       bool
	logFormat   string
	logLevel    string
	once        bool
	refreshRate time.Duration
	showVersion bool
	extraFeeds  config.FeedList
}

// defineFlags registers every command-line flag on fs. main registers them on flag.CommandLine,
// and the CLI reference test on a set of its own.
func defineFlags(fs *flag.FlagSet, f *cliFlags) {
	fs.StringVar(&f.cfgPath, "conf", "blackhole-threats.yaml", "Configuration file")
	fs.BoolVar(&f.checkConfig, "check-config", false, "Validate configuration and exit")
	fs.BoolVar(&f.debug, "debug", false, "Enable debug logging")
	fs.StringVar(&f.logFormat, "log-format", "logfmt", "Log format (logfmt or json)")
	fs.StringVar(&f.logLevel, "log-level", "info", "Log level (panic, fatal, error, warn, info, debug, trace)")
	fs.BoolVar(&f.once, "once", false, "Run a single refresh cycle and exit")
	fs.DurationVar(&f.refreshRate, "refresh-rate", 2*time.Hour, "Refresh timer")
	fs.BoolVar(&f.showVersion, "version", false, "Print version information and exit")
	fs.Var(&f.extraFeeds, "feed", "Threat intelligence feed (use multiple times)")
}

func main() {
	log.SetFormatter(logging.OperatorFormatter{})
	log.SetOutput(os.Stdout)

	var flags cliFlags
	defineFlags(flag.CommandLine, &flags)
	flag.Parse()

	if flags.showVersion {
		fmt.Printf("blackhole-threats %s\ncommit: %s\nbuilt: %s\n", buildinfo.DisplayVersion(), buildinfo.Commit, buildinfo.BuildDate)
		return
	}

	formatter, err := logging.NewFormatter(flags.logFormat)
	if err != nil {
		log.WithField("log_format", flags.logFormat).WithError(err).Fatal("Invalid log format")
	}
	log.SetFormatter(formatter)

	selectedLogLevel := flags.logLevel
	if flags.debug {
		selectedLogLevel = "debug"
	}

	parsedLogLevel, err := log.ParseLevel(selectedLogLevel)
	if err != nil {
		log.WithField("log_level", selectedLogLevel).WithError(err).Fatal("Invalid log level")
	}
	log.SetLevel(parsedLogLevel)

	cfg, err := config.Load(flags.cfgPath)
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}
	if err := cfg.Validate(); err != nil {
		log.WithError(err).Fatal("Invalid configuration")
	}

	feeds := append([]config.Feed{}, cfg.Feeds...)
	feeds = append(feeds, flags.extraFeeds...)
	if err := config.ValidateFeeds(feeds, cfg.GoBGP.Global.Config.As); err != nil {
		log.WithError(err).Fatal("Invalid feed configuration")
	}
	if flags.refreshRate <= 0 {
		log.WithField("refresh_interval", flags.refreshRate).Fatal("Refresh timer must be greater than zero")
	}
	if flags.checkConfig {
		fmt.Printf("configuration OK: %d feed(s)\n", len(feeds))
		return
	}
	logStartup(flags.cfgPath, cfg, len(cfg.Feeds), len(flags.extraFeeds), flags.refreshRate, flags.once)

	router, err := bgp.New(&cfg.GoBGP, cfg.Limits)
	if err != nil {
		log.WithError(err).Fatal("Failed to start BGP server")
	}

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGUSR1)
	defer signal.Stop(sigC)

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("Started blackhole-threats")
	if err := router.Run(ctx, feeds, flags.refreshRate, sigC, flags.once); err != nil {
		if stopErr := router.Stop(); stopErr != nil {
			log.WithError(stopErr).Warn("Failed to stop BGP server cleanly after runtime error")
		}
		log.WithError(err).Error("Route sync failed")
		os.Exit(1)
	}

	if ctx.Err() != nil {
		select {
		case sig := <-shutdownSignals:
			log.WithField("signal", sig.String()).Info("Shutdown requested")
		default:
			log.Info("Shutdown requested")
		}
	} else if flags.once {
		log.Info("Completed single refresh cycle")
	}

	if err := router.Stop(); err != nil {
		log.WithError(err).Warn("Failed to stop BGP server cleanly")
	} else {
		log.Info("Stopped blackhole-threats")
	}
}
