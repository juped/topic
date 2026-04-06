package git_test

import (
	"os/exec"
	"testing"

	"go.topic.tools/topic/internal/testutil"
	"go.topic.tools/topic/pkg/git"
)

func TestGitCommand(t *testing.T) {
	dir := testutil.SetupRepo(t)
	out, err := git.GitCommand(dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := git.GitRun(dir, "cat-file", "-e", out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitCommand_Failure(t *testing.T) {
	dir := testutil.SetupRepo(t)
	_, err := git.GitCommand(dir, "rev-parse", "nonexistent-ref")
	if err == nil {
		t.Error("expected error for nonexistent ref")
	}
}

func TestGitRun(t *testing.T) {
	dir := testutil.SetupRepo(t)
	if err := git.GitRun(dir, "status"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitPipe(t *testing.T) {
	dir := testutil.SetupRepo(t)
	out, err := git.GitPipe(dir, "hello", "hash-object", "-w", "--stdin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := git.GitRun(dir, "cat-file", "-e", out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitDir(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	gitDir, err := git.GitDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gitDir == "" {
		t.Error("expected non-empty git dir")
	}
}

func TestCommonGitDir(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	gitDir, err := git.CommonGitDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gitDir == "" {
		t.Error("expected non-empty git dir")
	}
}

func TestReleaseTag(t *testing.T) {
	dir := testutil.SetupRepo(t)
	tag, err := git.ReleaseTag(dir, "base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v0.0.0" {
		t.Errorf("expected v0.0.0, got %q", tag)
	}
}

func TestReleaseTag_NoBranch(t *testing.T) {
	dir := testutil.SetupRepo(t)
	_, err := git.ReleaseTag(dir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent branch")
	}
}

func TestBranchExists(t *testing.T) {
	dir := testutil.SetupRepo(t)
	if !git.BranchExists(dir, "base") {
		t.Error("base branch should exist")
	}
	if git.BranchExists(dir, "nonexistent") {
		t.Error("nonexistent branch should not exist")
	}
}

func TestRemoteDefaultBranch(t *testing.T) {
	remote := testutil.SetupRepo(t)
	local := t.TempDir()

	cmd := exec.Command("git", "clone", remote, ".")
	cmd.Dir = local
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}

	branch, err := git.RemoteDefaultBranch(local, "origin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "base" {
		t.Errorf("expected base, got %q", branch)
	}
}
