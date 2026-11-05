package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A word that isn't a flag used to end flag parsing without a word said: check-config without its
// dash started the daemon, and anything after a stray word was ignored. Both now fail like a bad
// flag does, with exit code 2. The timeout is for the old behaviour, where the first case runs
// the daemon until it's stopped.
func TestRejectsArgumentsThatAreNotFlags(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "blackhole-threats.yaml")
	config := `gobgp:
  global:
    config:
      as: 64520
      routerid: "198.51.100.10"
      port: -1
feeds: []
`
	if err := os.WriteFile(conf, []byte(config), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"-conf", conf, "check-config"}, `unexpected argument "check-config"; did you mean -check-config?`},
		{[]string{"-conf", conf, "-check-config", "stray", "-debug"}, `unexpected argument "stray"; blackhole-threats takes only flags`},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], tc.args...)
		cmd.Env = append(os.Environ(), runMainEnv+"=1")
		out, err := cmd.CombinedOutput()
		cancel()

		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 2 {
			t.Errorf("blackhole-threats %s: %v, want exit code 2; it printed:\n%s", strings.Join(tc.args[2:], " "), err, out)
			continue
		}
		if !strings.Contains(string(out), tc.want) || !strings.Contains(string(out), "-check-config") {
			t.Errorf("blackhole-threats %s printed:\n%s\nwant %q and the usage", strings.Join(tc.args[2:], " "), out, tc.want)
		}
	}
}
