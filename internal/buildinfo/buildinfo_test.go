package buildinfo

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestTagVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    string
	}{
		// What git describe gives, for a tag and for later or modified checkouts.
		{"v1.2.3", "v1.2.3"},
		{"v1.2.3-rc.4", "v1.2.3-rc.4"},
		{"v1.2.3-6-gfd35c0a", "v1.2.3"},
		{"v1.2.3-rc.4-6-gfd35c0a-dirty", "v1.2.3-rc.4"},
		// What debian/rules stamps: the changelog's upstream version, without the v, and with
		// a release candidate's -rc.N written ~rc.N by prepare-debian-changelog.sh.
		{"1.2.3", "v1.2.3"},
		{"1.2.3~rc.4", "v1.2.3-rc.4"},
		{" 1.2.3\n", "v1.2.3"},
		// Not releases.
		{"0.0.0~ci42", "unknown"},
		{"1.2.3-1", "unknown"},
		{"fd35c0a", "unknown"},
		{"dev", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		if got := tagVersion(tt.version); got != tt.want {
			t.Errorf("tagVersion(%q) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

// debian/rules stamps Version with the top debian/changelog entry's version, less the Debian
// revision. That has to be recognised as a release, or every Debian build logs
// tag_version=unknown. Build and Validate's package job rewrites the changelog to 0.0.0~ciN
// before dh_auto_test runs this, and that isn't a release, so only a release-shaped version is
// checked.
func TestTagVersionRecognisesTheDebianChangelogVersion(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "debian", "changelog"))
	if err != nil {
		t.Fatalf("read debian/changelog: %v", err)
	}
	first, _, _ := strings.Cut(string(data), "\n")
	match := regexp.MustCompile(`^\S+ \(([^)]+)\)`).FindStringSubmatch(first)
	if match == nil {
		t.Fatalf("can't find the version in debian/changelog's first line %q", first)
	}
	// The same as debian/rules: sed 's/-[^-]*$//'.
	upstream := match[1]
	if i := strings.LastIndex(upstream, "-"); i >= 0 {
		upstream = upstream[:i]
	}

	if !regexp.MustCompile(`^\d+\.\d+\.\d+(~rc\.\d+)?$`).MatchString(upstream) {
		t.Skipf("debian/changelog's version %q isn't a release, as in a CI package build", upstream)
	}

	if got, want := tagVersion(upstream), "v"+strings.Replace(upstream, "~rc.", "-rc.", 1); got != want {
		t.Fatalf("tagVersion(%q) = %q, want %q", upstream, got, want)
	}
}
