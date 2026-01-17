package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GitDir finds a git directory from the current working directory.
// In a worktree, this will actually find .git/worktrees/[name] in the
// real git directory. This is desirable because it applies per-worktree
// configuration to the git commands we call if this is on and it exists.
func GitDir() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	output, err := GitCommand(pwd, "rev-parse", "--path-format=absolute",
		"--git-dir")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// CommonGitDir finds *the* git directory from the current working directory.
// In a worktree, still gives the common git directory rather than the
// worktree-specific one. Usually we want that if it exists, but not
// for e.g. our own config file.
func CommonGitDir() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	output, err := GitCommand(pwd, "rev-parse", "--path-format=absolute",
		"--git-common-dir")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GitCommand runs a git command, given a git directory and args.
// Just a simple wrapper on exec, but we trim the newline off the end.
func GitCommand(gitDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = gitDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("git %s failed: %w\n%s",
			strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output)), err
}

// GitRun runs a git command, given a git directory and args,
// ignoring its output aside from exit status.
func GitRun(gitDir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = gitDir
	return cmd.Run()
}

// RemoteExists checks if a remote exists.
func RemoteExists(gitDir string, remote string) bool {
	return GitRun(gitDir, "remote", "get-url", remote) == nil
}

// BranchExists checks if a branch exists.
func BranchExists(gitDir string, branch string) bool {
	return GitRun("rev-parse", "--verify", branch) == nil
}

// RemoteDefaultBranch gets a default branch name from a remote.
// Returns a local branch name (e.g., "base"), not the remote-tracking
// branch name; we're just using the remote to learn a name.
func RemoteDefaultBranch(gitDir string, remote string) (string, error) {
	longRef, err := GitCommand(gitDir, "symbolic-ref", "--short",
		"refs/remotes/"+remote+"/HEAD")
	if err != nil {
		return "", err
	}

	shortRef, foundPrefix := strings.CutPrefix(string(longRef), remote+"/")
	if !foundPrefix {
		return "", errors.New(remote + "/HEAD didn't dereference as expected")
	}

	return shortRef, nil
}

// ReleaseTag finds a tag given a release branch.
// Tries annotated tags first, then any tags.
func ReleaseTag(gitDir string, branch string) (string, error) {
	tag, err := GitCommand(gitDir, "describe", "--abbrev=0", branch)
	if err != nil {
		tag, err = GitCommand(gitDir, "describe", "--abbrev=0", "--tags",
			branch)
		if err != nil {
			return "", err
		}
	}
	return string(tag), nil
}
