package testutil

import (
	"os/exec"
	"testing"
)

// SetupRepo creates a temporary git repository for use in tests.
// Initialized with default branch "base", user config, an initial empty
// commit, and an annotated "v0.0.0" tag. Returns the repo root path.
func SetupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	Run(t, dir, "git", "init", "-b", "base")
	Run(t, dir, "git", "config", "user.email", "test@example.com")
	Run(t, dir, "git", "config", "user.name", "Test")
	Run(t, dir, "git", "commit", "--allow-empty", "-m", "initial")
	Run(t, dir, "git", "tag", "-a", "v0.0.0", "-m", "v0.0.0")
	return dir
}

// Run executes a command in dir, calling t.Fatal if it exits non-zero.
func Run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v failed: %v\n%s", args, err, out)
	}
}