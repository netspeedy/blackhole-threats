package feed

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/netspeedy/blackhole-threats/internal/config"
)

var ErrUnhandledScheme = errors.New("unhandled scheme")

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

type Reader struct {
	Client *http.Client
	// Limits sets the shortest prefix kept for each address family. The zero value applies the
	// defaults, /8 for IPv4 and /16 for IPv6.
	Limits config.Limits
}

type ReadResult struct {
	Prefixes []netip.Prefix
	// Count is how many prefixes the feeds that were read gave, including any that were dropped.
	Count int
	// Dropped is how many of those were shorter than the limits and left out.
	Dropped       int
	FailedSources []string
}

type jsonFeedEntry struct {
	CIDR    string `json:"cidr"`
	Prefix  string `json:"prefix"`
	IP      string `json:"ip"`
	Address string `json:"address"`
}

func (r Reader) ReadMany(ctx context.Context, sources ...string) ReadResult {
	type result struct {
		source   string
		prefixes []netip.Prefix
		count    int
		err      error
	}

	results := make(chan result, len(sources))
	var wg sync.WaitGroup
	wg.Add(len(sources))

	for _, source := range sources {
		go func(source string) {
			defer wg.Done()

			prefixes, count, err := r.readOne(ctx, source)
			if err != nil {
				results <- result{source: source, err: err}
				return
			}

			results <- result{source: source, prefixes: prefixes, count: count}
		}(source)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var prefixes []netip.Prefix
	var total, dropped int
	var failed []string
	for result := range results {
		if result.err != nil {
			log.WithField("source", result.source).WithError(result.err).Error("Failed to read threat feed")
			failed = append(failed, result.source)
			continue
		}

		kept, short := r.dropShortPrefixes(result.prefixes, log.Fields{"source": result.source})
		prefixes = append(prefixes, kept...)
		total += result.count
		dropped += short
		log.WithFields(log.Fields{
			"source":           result.source,
			"prefixes_read":    result.count,
			"prefixes_dropped": short,
		}).Info("Parsed threat feed")
	}

	// Summarising never merges past the limits, so this shouldn't find anything. It's here so
	// that nothing shorter than the limits can be announced even if that ever changes.
	summarized, short := r.dropShortPrefixes(summarizePrefixes(prefixes, r.Limits), log.Fields{"sources": sources})
	dropped += short

	return ReadResult{
		Prefixes:      summarized,
		Count:         total,
		Dropped:       dropped,
		FailedSources: failed,
	}
}

// dropShortPrefixes returns the prefixes that are at least as long as the limit for their address
// family, and how many it left out. Each one left out is logged with fields, which say where it
// came from.
func (r Reader) dropShortPrefixes(prefixes []netip.Prefix, fields log.Fields) ([]netip.Prefix, int) {
	kept := make([]netip.Prefix, 0, len(prefixes))
	dropped := 0
	for _, prefix := range prefixes {
		limit := r.Limits.MinPrefixLength(prefix.Addr())
		if prefix.Bits() >= limit {
			kept = append(kept, prefix)
			continue
		}

		dropped++
		log.WithFields(fields).WithFields(log.Fields{
			"prefix":            prefix.String(),
			"min_prefix_length": limit,
		}).Warn("Dropped prefix shorter than the minimum length")
	}

	return kept, dropped
}

func (r Reader) readOne(ctx context.Context, source string) ([]netip.Prefix, int, error) {
	client := r.Client
	if client == nil {
		client = defaultHTTPClient
	}

	u, err := url.Parse(source)
	if err != nil {
		return nil, 0, err
	}

	switch u.Scheme {
	case "":
		handle, err := os.Open(source)
		if err != nil {
			return nil, 0, err
		}
		defer handle.Close()

		if isJSONFeed(source, "") {
			return readFromJSON(handle)
		}
		return readFromText(handle)
	case "http", "https":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, 0, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, 0, fmt.Errorf("non-OK status code: %d", resp.StatusCode)
		}

		if isJSONFeed(source, resp.Header.Get("Content-Type")) {
			return readFromJSON(resp.Body)
		}
		return readFromText(resp.Body)
	default:
		return nil, 0, fmt.Errorf("%w: %s", ErrUnhandledScheme, u.Scheme)
	}
}

func isJSONFeed(name, contentType string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".json", ".jsonl", ".ndjson":
		return true
	}

	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	mediaType = strings.ToLower(mediaType)

	switch mediaType {
	case "application/json", "application/jsonl", "application/ndjson", "application/x-ndjson", "text/json":
		return true
	default:
		return strings.HasSuffix(mediaType, "+json")
	}
}

func readFromText(r io.Reader) ([]netip.Prefix, int, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var prefixes []netip.Prefix
	for scanner.Scan() {
		prefix, ok := extractPrefix(scanner.Text())
		if !ok {
			continue
		}
		prefixes = append(prefixes, prefix)
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	return prefixes, len(prefixes), nil
}

func readFromJSON(r io.Reader) ([]netip.Prefix, int, error) {
	buffered := bufio.NewReader(r)
	lead, err := peekJSONLead(buffered)
	if err == io.EOF {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}

	if lead == '[' {
		return readFromJSONArray(buffered)
	}

	return readFromJSONStream(buffered)
}

func peekJSONLead(r *bufio.Reader) (byte, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		default:
			if err := r.UnreadByte(); err != nil {
				return 0, err
			}
			return b, nil
		}
	}
}

func readFromJSONStream(r io.Reader) ([]netip.Prefix, int, error) {
	decoder := json.NewDecoder(r)
	var prefixes []netip.Prefix

	for {
		var entry jsonFeedEntry
		err := decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		prefix, ok := prefixFromJSONEntry(entry)
		if !ok {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	return prefixes, len(prefixes), nil
}

func readFromJSONArray(r io.Reader) ([]netip.Prefix, int, error) {
	decoder := json.NewDecoder(r)
	token, err := decoder.Token()
	if err != nil {
		return nil, 0, err
	}

	delim, ok := token.(json.Delim)
	if !ok || delim != '[' {
		return nil, 0, fmt.Errorf("invalid JSON array feed")
	}

	var prefixes []netip.Prefix
	for decoder.More() {
		var entry jsonFeedEntry
		if err := decoder.Decode(&entry); err != nil {
			return nil, 0, err
		}

		prefix, ok := prefixFromJSONEntry(entry)
		if !ok {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	if _, err := decoder.Token(); err != nil {
		return nil, 0, err
	}

	return prefixes, len(prefixes), nil
}

func extractPrefix(line string) (netip.Prefix, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "//") {
		return netip.Prefix{}, false
	}

	for _, token := range strings.FieldsFunc(line, func(r rune) bool {
		switch r {
		case ' ', '\t', ',', ';':
			return true
		default:
			return false
		}
	}) {
		token = strings.Trim(token, "\"'()[]{}<>")
		if prefix, ok := parseNetworkToken(token); ok {
			return prefix, true
		}
	}

	return netip.Prefix{}, false
}

func prefixFromJSONEntry(entry jsonFeedEntry) (netip.Prefix, bool) {
	for _, candidate := range []string{entry.CIDR, entry.Prefix, entry.IP, entry.Address} {
		if prefix, ok := parseNetworkToken(candidate); ok {
			return prefix, true
		}
	}

	return netip.Prefix{}, false
}

func parseNetworkToken(token string) (netip.Prefix, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return netip.Prefix{}, false
	}

	if prefix, err := netip.ParsePrefix(token); err == nil {
		return prefix.Masked(), true
	}
	if addr, err := netip.ParseAddr(token); err == nil {
		return netip.PrefixFrom(addr, addr.BitLen()).Masked(), true
	}

	return netip.Prefix{}, false
}

// summarizePrefixes masks, deduplicates and merges prefixes into as few as cover them. It never
// merges two prefixes into a parent shorter than limits allows: two adjacent /8s stay as two /8s
// rather than becoming a /7 that would then have to be dropped, taking both with it.
func summarizePrefixes(prefixes []netip.Prefix, limits config.Limits) []netip.Prefix {
	if len(prefixes) == 0 {
		return nil
	}

	current := make([]netip.Prefix, 0, len(prefixes))
	for _, prefix := range prefixes {
		if prefix.IsValid() {
			current = append(current, prefix.Masked())
		}
	}
	if len(current) == 0 {
		return nil
	}

	sortPrefixes(current)
	current = pruneContainedPrefixes(current)

	for {
		next := mergeSiblingPrefixes(current, limits)
		sortPrefixes(next)
		next = pruneContainedPrefixes(next)
		if samePrefixes(current, next) {
			return next
		}
		current = next
	}
}

func sortPrefixes(prefixes []netip.Prefix) {
	sort.Slice(prefixes, func(i, j int) bool {
		a := prefixes[i]
		b := prefixes[j]
		if a.Addr() == b.Addr() {
			return a.Bits() < b.Bits()
		}
		return a.Addr().Less(b.Addr())
	})
}

func pruneContainedPrefixes(prefixes []netip.Prefix) []netip.Prefix {
	result := make([]netip.Prefix, 0, len(prefixes))
	for _, prefix := range prefixes {
		if len(result) == 0 {
			result = append(result, prefix)
			continue
		}

		last := result[len(result)-1]
		if last == prefix {
			continue
		}
		if last.Bits() <= prefix.Bits() && last.Contains(prefix.Addr()) {
			continue
		}

		result = append(result, prefix)
	}

	return result
}

func mergeSiblingPrefixes(prefixes []netip.Prefix, limits config.Limits) []netip.Prefix {
	result := make([]netip.Prefix, 0, len(prefixes))
	for _, prefix := range prefixes {
		if len(result) == 0 {
			result = append(result, prefix)
			continue
		}

		last := result[len(result)-1]
		if merged, ok := mergePair(last, prefix, limits); ok {
			result[len(result)-1] = merged
			continue
		}

		result = append(result, prefix)
	}

	return result
}

func mergePair(a, b netip.Prefix, limits config.Limits) (netip.Prefix, bool) {
	if a.Bits() != b.Bits() || a.Bits() == 0 || a.Addr().BitLen() != b.Addr().BitLen() {
		return netip.Prefix{}, false
	}
	if a.Bits()-1 < limits.MinPrefixLength(a.Addr()) {
		return netip.Prefix{}, false
	}

	parentA, ok := parentPrefix(a)
	if !ok {
		return netip.Prefix{}, false
	}
	parentB, ok := parentPrefix(b)
	if !ok || parentA != parentB {
		return netip.Prefix{}, false
	}

	return parentA, true
}

func parentPrefix(prefix netip.Prefix) (netip.Prefix, bool) {
	if !prefix.IsValid() || prefix.Bits() == 0 {
		return netip.Prefix{}, false
	}

	return netip.PrefixFrom(prefix.Masked().Addr(), prefix.Bits()-1).Masked(), true
}

func samePrefixes(a, b []netip.Prefix) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
