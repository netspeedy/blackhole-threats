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

// ErrNoPrefixes means a feed was read without an error but gave nothing to announce: an empty
// response, an HTML page, an empty JSON array, a JSON object wrapping the list, or only prefixes
// shorter than the limits. Unless the feed sets allow_empty, that counts as a failure, so its community keeps the
// routes it had rather than withdrawing them all because of a broken response.
var ErrNoPrefixes = errors.New("no prefixes in feed")

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

// ReadMany reads every feed at the same time and summarises what they give together. A feed that
// can't be read, or gives no prefixes and doesn't set AllowEmpty, is listed in FailedSources and
// contributes nothing.
func (r Reader) ReadMany(ctx context.Context, feeds ...config.Feed) ReadResult {
	type result struct {
		feed     config.Feed
		prefixes []netip.Prefix
		count    int
		err      error
	}

	sources := make([]string, len(feeds))
	results := make(chan result, len(feeds))
	var wg sync.WaitGroup
	wg.Add(len(feeds))

	for i, feed := range feeds {
		sources[i] = feed.URL
		go func(feed config.Feed) {
			defer wg.Done()

			prefixes, count, err := r.readOne(ctx, feed.URL)
			if errors.Is(err, ErrNoPrefixes) && feed.AllowEmpty {
				err = nil
			}
			if err != nil {
				results <- result{feed: feed, err: err}
				return
			}

			results <- result{feed: feed, prefixes: prefixes, count: count}
		}(feed)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var prefixes []netip.Prefix
	var total, dropped int
	var failed []string
	for result := range results {
		source := result.feed.URL
		if result.err != nil {
			log.WithField("source", source).WithError(result.err).Error("Failed to read threat feed")
			failed = append(failed, source)
			continue
		}

		kept, short := r.dropShortPrefixes(result.prefixes, log.Fields{"source": source})
		if len(kept) == 0 && short > 0 && !result.feed.AllowEmpty {
			err := fmt.Errorf("%w: all %d were shorter than the limits", ErrNoPrefixes, short)
			log.WithField("source", source).WithError(err).Error("Failed to read threat feed")
			failed = append(failed, source)
			continue
		}

		prefixes = append(prefixes, kept...)
		total += result.count
		dropped += short
		log.WithFields(log.Fields{
			"source":           source,
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

		return readFeed(handle, "file", isJSONFeed(source, ""), "")
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

		contentType := resp.Header.Get("Content-Type")
		return readFeed(resp.Body, "response", isJSONFeed(source, contentType), contentType)
	default:
		return nil, 0, fmt.Errorf("%w: %s", ErrUnhandledScheme, u.Scheme)
	}
}

// readFeed reads r as JSON or text. If it gives no prefixes, the error wraps ErrNoPrefixes and
// says what was there instead, so the log line shows why: what is either "file" or "response",
// and contentType is the response's Content-Type, if any.
func readFeed(r io.Reader, what string, asJSON bool, contentType string) ([]netip.Prefix, int, error) {
	counted := &countingReader{r: r}

	var prefixes []netip.Prefix
	var shape jsonShape
	var err error
	if asJSON {
		prefixes, shape, err = readFromJSON(counted)
	} else {
		prefixes, _, err = readFromText(counted)
	}
	if err != nil || len(prefixes) > 0 {
		return prefixes, len(prefixes), err
	}

	var detail string
	switch {
	case counted.n == 0:
		detail = fmt.Sprintf("the %s was empty", what)
	case asJSON && shape.array && shape.entries == 0:
		detail = fmt.Sprintf("the %s was an empty JSON array", what)
	case asJSON:
		detail = fmt.Sprintf("the %s was %d bytes of JSON with no object holding an address in cidr, prefix, ip or address", what, counted.n)
	default:
		detail = fmt.Sprintf("the %s was %d bytes of text with no address or prefix on any line", what, counted.n)
	}
	if contentType != "" {
		detail += fmt.Sprintf(" (Content-Type %q)", contentType)
	}

	return nil, 0, fmt.Errorf("%w: %s", ErrNoPrefixes, detail)
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
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

// jsonShape is what a JSON feed held, so that one giving no prefixes can be told apart: an empty
// array is a list with nothing in it, where a wrapper object or entries with other field names
// mean the format isn't one the reader knows.
type jsonShape struct {
	// array is true when the feed was a JSON array rather than a stream of values.
	array bool
	// entries is how many values the array or the stream held.
	entries int
}

func readFromJSON(r io.Reader) ([]netip.Prefix, jsonShape, error) {
	buffered := bufio.NewReader(r)
	lead, err := peekJSONLead(buffered)
	if err == io.EOF {
		return nil, jsonShape{}, nil
	}
	if err != nil {
		return nil, jsonShape{}, err
	}

	if lead == '[' {
		prefixes, entries, err := readFromJSONArray(buffered)
		return prefixes, jsonShape{array: true, entries: entries}, err
	}

	prefixes, entries, err := readFromJSONStream(buffered)
	return prefixes, jsonShape{entries: entries}, err
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

// readFromJSONStream reads a stream of JSON objects, such as JSON Lines, and returns the prefixes
// in them and how many objects there were.
func readFromJSONStream(r io.Reader) ([]netip.Prefix, int, error) {
	decoder := json.NewDecoder(r)
	var prefixes []netip.Prefix
	entries := 0

	for {
		var entry jsonFeedEntry
		err := decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		entries++

		prefix, ok := prefixFromJSONEntry(entry)
		if !ok {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	return prefixes, entries, nil
}

// readFromJSONArray reads a JSON array of objects and returns the prefixes in them and how many
// objects there were.
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
	entries := 0
	for decoder.More() {
		var entry jsonFeedEntry
		if err := decoder.Decode(&entry); err != nil {
			return nil, 0, err
		}
		entries++

		prefix, ok := prefixFromJSONEntry(entry)
		if !ok {
			continue
		}
		prefixes = append(prefixes, prefix)
	}

	if _, err := decoder.Token(); err != nil {
		return nil, 0, err
	}

	return prefixes, entries, nil
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
