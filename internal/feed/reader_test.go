package feed

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/netspeedy/blackhole-threats/internal/config"
)

const sampleJSONLFeed = `{"cidr":"203.0.113.0/24","rir":"TEST","sblid":"SBL1"}
{"cidr":"2001:db8::/32","rir":"TEST","sblid":"SBL2"}
{"generated_at":"2026-04-19T00:00:00Z"}`

const sampleJSONArrayFeed = `[
  {"cidr":"203.0.113.0/24","rir":"TEST","sblid":"SBL1"},
  {"cidr":"2001:db8::/32","rir":"TEST","sblid":"SBL2"},
  {"generated_at":"2026-04-19T00:00:00Z"}
]`

func TestReadFromJSON(t *testing.T) {
	t.Parallel()

	prefixes, count, err := readFromJSON(strings.NewReader(sampleJSONLFeed))
	if err != nil {
		t.Fatalf("readFromJSON() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readFromJSON() count = %d, want %d", count, 2)
	}

	want := []string{"203.0.113.0/24", "2001:db8::/32"}
	if got := prefixesToStrings(prefixes); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("readFromJSON() nets = %v, want %v", got, want)
	}
}

func TestReaderReadsJSONLFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	feedFile := filepath.Join(dir, "feed.jsonl")
	if err := os.WriteFile(feedFile, []byte(sampleJSONLFeed), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	reader := Reader{}
	_, count, err := reader.readOne(context.Background(), feedFile)
	if err != nil {
		t.Fatalf("readOne() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readOne() count = %d, want %d", count, 2)
	}
}

func TestReaderReadsNDJSONOverHTTP(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, sampleJSONLFeed)
	}))
	defer srv.Close()

	reader := Reader{Client: srv.Client()}
	_, count, err := reader.readOne(context.Background(), srv.URL+"/feed.txt")
	if err != nil {
		t.Fatalf("readOne() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readOne() count = %d, want %d", count, 2)
	}
}

func TestReadFromJSONArray(t *testing.T) {
	t.Parallel()

	prefixes, count, err := readFromJSON(strings.NewReader(sampleJSONArrayFeed))
	if err != nil {
		t.Fatalf("readFromJSON() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readFromJSON() count = %d, want %d", count, 2)
	}

	want := []string{"203.0.113.0/24", "2001:db8::/32"}
	if got := prefixesToStrings(prefixes); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("readFromJSON() nets = %v, want %v", got, want)
	}
}

func TestReaderReadManyTracksFailedSources(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	goodFeed := filepath.Join(dir, "good.txt")
	if err := os.WriteFile(goodFeed, []byte("198.51.100.0/24\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := Reader{}.ReadMany(context.Background(), goodFeed, filepath.Join(dir, "missing.txt"))

	if result.Count != 1 {
		t.Fatalf("ReadMany() count = %d, want %d", result.Count, 1)
	}
	if len(result.Prefixes) != 1 || result.Prefixes[0].String() != "198.51.100.0/24" {
		t.Fatalf("ReadMany() prefixes = %v, want [198.51.100.0/24]", prefixesToStrings(result.Prefixes))
	}
	if len(result.FailedSources) != 1 {
		t.Fatalf("ReadMany() failed sources = %v, want 1 failure", result.FailedSources)
	}
}

func TestReadFromTextAcceptsIPsAndCIDRs(t *testing.T) {
	t.Parallel()

	text := strings.NewReader("203.0.113.10\n# comment\n198.51.100.0/24\n")
	prefixes, count, err := readFromText(text)
	if err != nil {
		t.Fatalf("readFromText() error = %v", err)
	}

	if count != 2 {
		t.Fatalf("readFromText() count = %d, want %d", count, 2)
	}

	want := []string{"203.0.113.10/32", "198.51.100.0/24"}
	if got := prefixesToStrings(prefixes); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("readFromText() prefixes = %v, want %v", got, want)
	}
}

func TestSummarizePrefixesMergesSiblings(t *testing.T) {
	t.Parallel()

	prefixes := []netip.Prefix{
		netip.MustParsePrefix("198.51.100.0/25"),
		netip.MustParsePrefix("198.51.100.128/25"),
		netip.MustParsePrefix("2001:db8::/33"),
		netip.MustParsePrefix("2001:db8:8000::/33"),
	}

	want := []string{"198.51.100.0/24", "2001:db8::/32"}
	if got := prefixesToStrings(summarizePrefixes(prefixes, config.Limits{})); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("summarizePrefixes() = %v, want %v", got, want)
	}
}

func TestReadManyDropsDefaultRoutes(t *testing.T) {
	t.Parallel()

	source := writeFeed(t, "feed.txt", "0.0.0.0/0\n198.51.100.0/24\n::/0\n2001:db8::/32\n")
	result := Reader{}.ReadMany(context.Background(), source)

	assertPrefixes(t, result.Prefixes, "198.51.100.0/24", "2001:db8::/32")
	if result.Count != 4 || result.Dropped != 2 {
		t.Fatalf("ReadMany() count = %d, dropped = %d, want 4 and 2", result.Count, result.Dropped)
	}
	if len(result.FailedSources) != 0 {
		t.Fatalf("ReadMany() failed sources = %v, want none: a dropped line mustn't fail the feed", result.FailedSources)
	}
}

func TestReadManyDropsPrefixesShorterThanTheDefaults(t *testing.T) {
	t.Parallel()

	// One either side of each default: /7 and /8 for IPv4, /15 and /16 for IPv6.
	source := writeFeed(t, "feed.txt", "10.0.0.0/7\n12.0.0.0/8\n2000::/15\n2400::/16\n")
	result := Reader{}.ReadMany(context.Background(), source)

	assertPrefixes(t, result.Prefixes, "12.0.0.0/8", "2400::/16")
	if result.Dropped != 2 {
		t.Fatalf("ReadMany() dropped = %d, want 2", result.Dropped)
	}
}

func TestReadManyAppliesConfiguredLimits(t *testing.T) {
	t.Parallel()

	source := writeFeed(t, "feed.json", `[
  {"cidr":"192.0.2.0/23"},
  {"cidr":"198.51.100.0/24"},
  {"cidr":"2001:db8::/47"},
  {"cidr":"2001:db8:10::/48"}
]`)
	reader := Reader{Limits: config.Limits{MinIPv4PrefixLength: intPtr(24), MinIPv6PrefixLength: intPtr(48)}}
	result := reader.ReadMany(context.Background(), source)

	assertPrefixes(t, result.Prefixes, "198.51.100.0/24", "2001:db8:10::/48")
	if result.Count != 4 || result.Dropped != 2 {
		t.Fatalf("ReadMany() count = %d, dropped = %d, want 4 and 2", result.Count, result.Dropped)
	}
}

// Two neighbouring prefixes at the limit would summarise into their parent, one bit shorter than
// the limit. Summarising has to stop there and keep both, rather than produce the parent or drop
// it along with both of them.
func TestReadManyDoesNotSummariseBelowTheLimits(t *testing.T) {
	t.Parallel()

	source := writeFeed(t, "feed.txt", "10.0.0.0/8\n11.0.0.0/8\n2000::/16\n2001::/16\n")
	result := Reader{}.ReadMany(context.Background(), source)
	assertPrefixes(t, result.Prefixes, "10.0.0.0/8", "11.0.0.0/8", "2000::/16", "2001::/16")
	if result.Dropped != 0 {
		t.Fatalf("ReadMany() dropped = %d, want 0", result.Dropped)
	}

	custom := writeFeed(t, "custom.txt", "192.0.2.0/24\n192.0.3.0/24\n198.51.100.0/25\n198.51.100.128/25\n")
	reader := Reader{Limits: config.Limits{MinIPv4PrefixLength: intPtr(24)}}
	assertPrefixes(t, reader.ReadMany(context.Background(), custom).Prefixes, "192.0.2.0/24", "192.0.3.0/24", "198.51.100.0/24")
}

// The check after summarising is a backstop: summarising stops at the limits, so it shouldn't
// find anything, but if it does the prefix is still dropped.
func TestDropShortPrefixesCatchesSummarisedPrefixes(t *testing.T) {
	t.Parallel()

	summarized := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/7"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("2000::/15"),
	}
	kept, dropped := Reader{}.dropShortPrefixes(summarized, log.Fields{"sources": []string{"test"}})
	assertPrefixes(t, kept, "198.51.100.0/24")
	if dropped != 2 {
		t.Fatalf("dropShortPrefixes() dropped = %d, want 2", dropped)
	}
}

// Not parallel: it swaps the standard logger's output to read the warning back.
func TestReadManyLogsEachDroppedPrefix(t *testing.T) {
	var buf bytes.Buffer
	logger := log.StandardLogger()
	out, formatter := logger.Out, logger.Formatter
	logger.SetOutput(&buf)
	logger.SetFormatter(&log.JSONFormatter{})
	t.Cleanup(func() {
		logger.SetOutput(out)
		logger.SetFormatter(formatter)
	})

	source := writeFeed(t, "feed.txt", "0.0.0.0/0\n198.51.100.0/24\n")
	Reader{}.ReadMany(context.Background(), source)

	var warning, parsed map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("log line %q isn't JSON: %v", line, err)
		}
		switch entry["msg"] {
		case "Dropped prefix shorter than the minimum length":
			warning = entry
		case "Parsed threat feed":
			parsed = entry
		}
	}

	if warning == nil {
		t.Fatalf("no warning for the dropped prefix in:\n%s", buf.String())
	}
	if warning["level"] != "warning" || warning["source"] != source || warning["prefix"] != "0.0.0.0/0" || warning["min_prefix_length"] != float64(8) {
		t.Fatalf("warning = %v, want level warning, source %s, prefix 0.0.0.0/0 and min_prefix_length 8", warning, source)
	}
	if parsed == nil || parsed["prefixes_read"] != float64(2) || parsed["prefixes_dropped"] != float64(1) {
		t.Fatalf("Parsed threat feed = %v, want prefixes_read 2 and prefixes_dropped 1", parsed)
	}
}

func writeFeed(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func assertPrefixes(t *testing.T, got []netip.Prefix, want ...string) {
	t.Helper()

	if values := prefixesToStrings(got); !slices.Equal(values, want) {
		t.Fatalf("prefixes = %v, want %v", values, want)
	}
}

func intPtr(v int) *int {
	return &v
}

func prefixesToStrings(prefixes []netip.Prefix) []string {
	values := make([]string, len(prefixes))
	for i, prefix := range prefixes {
		values[i] = prefix.String()
	}
	return values
}
