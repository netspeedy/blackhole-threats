package buildinfo

import (
	"regexp"
	"strings"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// releaseTagPattern matches the version a release build is stamped with. That's the tag as git
// describe prints it, v1.2.3 or v1.2.3-rc.4, with -N-gHASH and -dirty after it for a later or
// modified checkout. A Debian build is stamped from debian/changelog instead, which has no v and
// writes a release candidate as 1.2.3~rc.4, so the v is optional and ~ stands in for the -.
var releaseTagPattern = regexp.MustCompile(`^v?(\d+\.\d+\.\d+)(?:[-~]rc\.(\d+))?(?:-\d+-g[0-9a-f]+)?(?:-dirty)?$`)

func DisplayVersion() string {
	switch {
	case Version != "" && Version != "dev" && Version != "main" && Version != "master":
		return Version
	case Commit != "" && Commit != "unknown":
		return ShortCommit()
	case Version != "":
		return Version
	default:
		return "unknown"
	}
}

func ShortCommit() string {
	commit := strings.TrimSpace(Commit)
	if commit == "" || commit == "unknown" {
		return "unknown"
	}

	if len(commit) > 7 {
		return commit[:7]
	}

	return commit
}

// TagVersion returns the release tag the binary was built from, such as v1.2.3 or v1.2.3-rc.4, or
// "unknown" when Version isn't a release.
func TagVersion() string {
	return tagVersion(Version)
}

func tagVersion(version string) string {
	match := releaseTagPattern.FindStringSubmatch(strings.TrimSpace(version))
	if match == nil {
		return "unknown"
	}

	tag := "v" + match[1]
	if match[2] != "" {
		tag += "-rc." + match[2]
	}
	return tag
}
