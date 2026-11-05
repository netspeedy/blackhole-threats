package main

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The website's CLI reference is written by hand so flags can be grouped and explained, which
// means nothing regenerates it when a flag changes. This keeps the page and the CLI in step:
// every flag must be documented, and every documented flag must exist. Flags take one dash, as in
// the man page, so the page lists them as `flag: "-conf"`.
func TestCLIReferenceCoversEveryFlag(t *testing.T) {
	page := filepath.Join("..", "..", "website", "src", "content", "docs", "reference", "cli.mdx")
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatalf("read the CLI reference: %v", err)
	}
	doc := string(data)
	fs := flag.NewFlagSet("blackhole-threats", flag.ContinueOnError)
	defineFlags(fs, &cliFlags{})

	fs.VisitAll(func(f *flag.Flag) {
		if !strings.Contains(doc, `flag: "-`+f.Name+`"`) {
			t.Errorf("%s doesn't document -%s", page, f.Name)
		}
	})

	for _, match := range regexp.MustCompile(`flag: "-([a-z][a-z0-9-]*)"`).FindAllStringSubmatch(doc, -1) {
		if fs.Lookup(match[1]) == nil {
			t.Errorf("%s documents -%s, which blackhole-threats doesn't accept", page, match[1])
		}
	}
}
