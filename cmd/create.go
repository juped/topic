package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"go.topic.tools/topic/pkg/git"
	"go.topic.tools/topic/pkg/topic"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [topic name]",
	Short: "Create a topic",
	Long:  `Create a topic at the current base point, optionally with dependencies.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := topicCreate(cmd, args); err != nil {
			return err
		}
		return nil
	},
}

var dependencies []string

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringArrayVarP(
		&dependencies,
		"depends", "d",
		[]string{},
		`The name of another topic this topic depends on.
(Can be repeated for multiple dependencies.)`)
}

func topicCreate(cmd *cobra.Command, args []string) error {
	config, err := topic.LoadConfig()
	if err != nil {
		return err
	}

	if config.BaseBranch == "" {
		return errors.New("don't know the base branch")
	}

	gitDir, err := git.GitDir()
	if err != nil {
		return err
	}

	basePoint, err := git.ReleaseTag(gitDir, config.BaseBranch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "release tag not found, "+
			"basing on %s instead\n", config.BaseBranch)
		fmt.Fprintf(os.Stderr, "remember to make a release tag asap!\n")
		basePoint = config.BaseBranch
	}

	if len(dependencies) == 0 {
		err = git.GitRun(gitDir, "branch", args[0], basePoint)
		if err != nil {
			return err
		}
	} else if len(dependencies) == 1 {
		fmt.Fprintf(os.Stderr, "only one dependency, basing directly on it\n")
		err = git.GitRun(gitDir, "branch", args[0], dependencies[0])
		if err != nil {
			return err
		}
	} else {
		fmt.Fprintf(os.Stderr, "multiple dependencies, merging them\n")
		octopusBase, err := octopusMergeDependencies(gitDir, basePoint,
			dependencies)
		if err != nil {
			return err
		}
		err = git.GitRun(gitDir, "branch", args[0], octopusBase)
		if err != nil {
			return err
		}
	}

	fmt.Printf("created topic branch %s based on %s", args[0], basePoint)
	if len(dependencies) > 0 {
		fmt.Printf(" and %d dependencies", len(dependencies))
	}
	fmt.Printf("\n")
	return nil
}

func octopusMergeDependencies(gitDir string, basePoint string,
	dependencies []string) (string, error) {
	accumulator := basePoint
	parents := []string{basePoint}

	for _, dep := range dependencies {
		tree, err := git.GitCommand(gitDir, "merge-tree", accumulator, dep)
		if err != nil {
			return "", err
		}

		parents = append(parents, dep)

		commitTreeArgs := []string{"commit-tree", tree, "-m", "temporary"}
		for _, parent := range parents {
			commitTreeArgs = append(commitTreeArgs, "-p", parent)
		}
		commit, err := git.GitCommand(gitDir, commitTreeArgs...)
		if err != nil {
			return "", err
		}
		accumulator = commit
	}

	mergeMessage, err := fmtMergeMsg(gitDir, dependencies)
	if err != nil {
		return "", err
	}

	commitTreeArgs := []string{"commit-tree", accumulator + "^{tree}"}
	for _, parent := range parents {
		commitTreeArgs = append(commitTreeArgs, "-p", parent)
	}
	commitTreeArgs = append(commitTreeArgs, "-m", mergeMessage)

	commit, err := git.GitCommand(gitDir, commitTreeArgs...)
	if err != nil {
		return "", err
	}

	return commit, nil
}

func fmtMergeMsg(gitDir string, dependencies []string) (string, error) {
	var fetchHeadLines []string
	for _, dep := range dependencies {
		sha, err := git.GitCommand(gitDir, "rev-parse", dep)
		if err != nil {
			return "", err
		}
		fetchHeadLines = append(fetchHeadLines, sha+"\t\tbranch '"+dep+"'")
	}
	fetchHead := strings.Join(fetchHeadLines, "\n") + "\n"

	msg, err := git.GitPipe(gitDir, fetchHead, "fmt-merge-msg")
	if err != nil {
		return "", err
	}

	// Rename the subject line: "branches"/"branch" -> "dependencies"/"dependency"
	lines := strings.SplitN(msg, "\n", 2)
	lines[0] = strings.Replace(lines[0], "branches", "dependencies", 1)
	lines[0] = strings.Replace(lines[0], "branch", "dependency", 1)
	return strings.Join(lines, "\n"), nil
}
