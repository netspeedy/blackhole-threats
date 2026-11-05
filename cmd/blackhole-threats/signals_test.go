package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// runMainEnv makes the test binary run main() instead of the tests, so a test can start the real
// daemon as a child process and signal it.
const runMainEnv = "BLACKHOLE_THREATS_TEST_RUN_MAIN"

func TestMain(m *testing.M) {
	if os.Getenv(runMainEnv) == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestDaemonSignals runs the daemon with GoBGP not listening and no neighbors, so it touches no
// network, and checks what each signal does: SIGUSR1 and SIGHUP refresh and leave it running, and
// SIGTERM stops it cleanly. SIGHUP used to end the process at once, skipping the shutdown.
func TestDaemonSignals(t *testing.T) {
	dir := t.TempDir()
	feed := filepath.Join(dir, "feed.txt")
	if err := os.WriteFile(feed, []byte("198.51.100.0/24\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	conf := filepath.Join(dir, "blackhole-threats.yaml")
	config := `gobgp:
  global:
    config:
      as: 64520
      routerid: "198.51.100.10"
      port: -1
feeds:
  - url: ` + feed + `
    community: 64520:1200
`
	if err := os.WriteFile(conf, []byte(config), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := exec.Command(os.Args[0], "-conf", conf)
	cmd.Env = append(os.Environ(), runMainEnv+"=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the daemon: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	lines := make(chan string, 100)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	var seen []string
	waitFor := func(want string) {
		t.Helper()
		timeout := time.After(10 * time.Second)
		for {
			select {
			case line, ok := <-lines:
				if !ok {
					err := cmd.Wait()
					t.Fatalf("the daemon stopped (%v) before logging %s; it logged:\n%s", err, want, strings.Join(seen, "\n"))
				}
				seen = append(seen, line)
				if strings.Contains(line, want) {
					return
				}
			case <-timeout:
				t.Fatalf("no %s within 10s; the daemon logged:\n%s", want, strings.Join(seen, "\n"))
			}
		}
	}

	waitFor(`msg="Refresh completed"`)
	for _, sig := range []struct {
		signal os.Signal
		name   string
	}{
		{syscall.SIGUSR1, `signal="user defined signal 1"`},
		{syscall.SIGHUP, `signal=hangup`},
	} {
		if err := cmd.Process.Signal(sig.signal); err != nil {
			t.Fatalf("send %v: %v", sig.signal, err)
		}
		waitFor(`msg="Refresh signal received" ` + sig.name)
		waitFor(`msg="Refresh completed"`)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}
	waitFor(`msg="Shutdown requested" signal=terminated`)
	waitFor(`msg="Stopped blackhole-threats"`)

	if err := cmd.Wait(); err != nil {
		t.Fatalf("the daemon exited with %v after SIGTERM, want exit code 0", err)
	}
}
