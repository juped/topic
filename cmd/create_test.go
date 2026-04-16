package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"go.topic.tools/topic/internal/testutil"
	"go.topic.tools/topic/pkg/git"
	"go.topic.tools/topic/pkg/topic"
)

// writeTopicConfig writes a minimal .git/topic/config for tests.
func writeTopicConfig(t *testing.T, dir string) {
	t.Helper()
	configDir := filepath.Join(dir, ".git", "topic")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "[topic]\nbaseBranch = base\nsync = false\n"
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// createBranch creates a branch with a single file commit, for use as a dependency.
// Leaves the repo back on "base" when done.
func createBranch(t *testing.T, dir, branch, filename string) {
	t.Helper()
	testutil.Run(t, dir, "git", "checkout", "-b", branch)
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(branch+"\n"), 0644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
	testutil.Run(t, dir, "git", "add", filename)
	testutil.Run(t, dir, "git", "commit", "-m", "add "+filename)
	testutil.Run(t, dir, "git", "checkout", "base")
}

// resetDependencies clears the global flag slice before and after a test.
// Necessary because cobra's StringArrayVar appends rather than resetting
// between Execute calls.
func resetDependencies(t *testing.T) {
	t.Helper()
	dependencies = []string{}
	t.Cleanup(func() { dependencies = []string{} })
}

// resetBugfixRev clears the global bugfixRev flag before and after a test.
func resetBugfixRev(t *testing.T) {
	t.Helper()
	bugfixRev = ""
	t.Cleanup(func() { bugfixRev = "" })
}

func TestFmtMergeMsg(t *testing.T) {
	dir := testutil.SetupRepo(t)
	createBranch(t, dir, "dep-a", "a.txt")
	createBranch(t, dir, "dep-b", "b.txt")
	t.Chdir(dir)

	gitDir, err := git.GitDir()
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}

	msg, err := fmtMergeMsg(gitDir, []string{"dep-a", "dep-b"})
	if err != nil {
		t.Fatalf("fmtMergeMsg: %v", err)
	}
	if msg == "" {
		t.Error("expected non-empty merge message")
	}
}

func TestOctopusMergeDependencies(t *testing.T) {
	dir := testutil.SetupRepo(t)
	createBranch(t, dir, "dep-a", "a.txt")
	createBranch(t, dir, "dep-b", "b.txt")
	t.Chdir(dir)

	gitDir, err := git.GitDir()
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}

	commit, err := octopusMergeDependencies(gitDir, "v0.0.0", []string{"dep-a", "dep-b"})
	if err != nil {
		t.Fatalf("octopusMergeDependencies: %v", err)
	}
	if err := git.GitRun(dir, "cat-file", "-e", commit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTopicCreate_NoDeps(t *testing.T) {
	dir := testutil.SetupRepo(t)
	writeTopicConfig(t, dir)
	t.Chdir(dir)
	resetDependencies(t)

	rootCmd.SetArgs([]string{"create", "my-feature"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create: %v", err)
	}

	gitDir := filepath.Join(dir, ".git")
	if _, err := git.GitCommand(gitDir, "rev-parse", "--verify", "my-feature"); err != nil {
		t.Error("branch my-feature should exist after create")
	}
}

func TestTopicCreate_OneDep(t *testing.T) {
	dir := testutil.SetupRepo(t)
	writeTopicConfig(t, dir)
	createBranch(t, dir, "dep-a", "a.txt")
	t.Chdir(dir)
	resetDependencies(t)

	rootCmd.SetArgs([]string{"create", "my-feature", "--depends", "dep-a"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create: %v", err)
	}

	// With a single dep, the branch should point to the same commit as dep-a.
	gitDir := filepath.Join(dir, ".git")
	depSHA, err := git.GitCommand(gitDir, "rev-parse", "dep-a")
	if err != nil {
		t.Fatalf("rev-parse dep-a: %v", err)
	}
	featureSHA, err := git.GitCommand(gitDir, "rev-parse", "my-feature")
	if err != nil {
		t.Fatalf("rev-parse my-feature: %v", err)
	}
	if depSHA != featureSHA {
		t.Error("my-feature should point to the same commit as dep-a")
	}
}

func TestTopicCreate_Bugfix(t *testing.T) {
	dir := testutil.SetupRepo(t)
	writeTopicConfig(t, dir)
	t.Chdir(dir)
	resetDependencies(t)
	resetBugfixRev(t)

	// Grab the current HEAD commit to use as the bugfix rev.
	gitDir := filepath.Join(dir, ".git")
	headSHA, err := git.GitCommand(gitDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}

	rootCmd.SetArgs([]string{"create", "my-bugfix", "--bugfix", headSHA})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create --bugfix: %v", err)
	}

	branchSHA, err := git.GitCommand(gitDir, "rev-parse", "my-bugfix")
	if err != nil {
		t.Error("branch my-bugfix should exist after create --bugfix")
	}
	if branchSHA != headSHA {
		t.Errorf("my-bugfix should point to %s, got %s", headSHA, branchSHA)
	}
}

func TestTopicCreate_BugfixWithDepsDisallowed(t *testing.T) {
	dir := testutil.SetupRepo(t)
	writeTopicConfig(t, dir)
	createBranch(t, dir, "dep-a", "a.txt")
	t.Chdir(dir)
	resetDependencies(t)
	resetBugfixRev(t)

	gitDir := filepath.Join(dir, ".git")
	headSHA, err := git.GitCommand(gitDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}

	rootCmd.SetArgs([]string{"create", "my-bugfix", "--bugfix", headSHA, "--depends", "dep-a"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error when combining --bugfix and --depends")
	}
}

func TestTopicCreate_CreatesMetadata(t *testing.T) {
	dir := testutil.SetupRepo(t)
	writeTopicConfig(t, dir)
	t.Chdir(dir)
	resetDependencies(t)

	rootCmd.SetArgs([]string{"create", "my-feature"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create: %v", err)
	}

	cfg, err := topic.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	metadata, err := topic.LoadMetadata(cfg)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	tm := topic.GetTopicByName(metadata, "my-feature")
	if tm == nil {
		t.Fatal("topic my-feature not found in metadata after create")
	}

	gitDir := filepath.Join(dir, ".git")
	wantHash, err := git.GitCommand(gitDir, "rev-parse", "my-feature")
	if err != nil {
		t.Fatalf("rev-parse my-feature: %v", err)
	}
	if tm.Hash != wantHash {
		t.Errorf("Hash: got %q, want %q", tm.Hash, wantHash)
	}
	if tm.Revision != 1 {
		t.Errorf("Revision: got %d, want 1", tm.Revision)
	}
}

func TestTopicCreate_MultiDeps(t *testing.T) {
	dir := testutil.SetupRepo(t)
	writeTopicConfig(t, dir)
	createBranch(t, dir, "dep-a", "a.txt")
	createBranch(t, dir, "dep-b", "b.txt")
	t.Chdir(dir)
	resetDependencies(t)

	rootCmd.SetArgs([]string{"create", "my-feature", "--depends", "dep-a", "--depends", "dep-b"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create: %v", err)
	}

	gitDir := filepath.Join(dir, ".git")
	if _, err := git.GitCommand(gitDir, "rev-parse", "--verify", "my-feature"); err != nil {
		t.Error("branch my-feature should exist after create with multiple deps")
	}
}
