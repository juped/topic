package cmd

import (
	"errors"
	"fmt"
	"os"
	"topic/pkg/git"
	"topic/pkg/topic"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [topic name]",
	Short: "Create a topic",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := topicCreate(cmd, args); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
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

	releaseTag, err := git.ReleaseTag(gitDir, config.BaseBranch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "release tag not found, "+
			"basing on %s instead\n", config.BaseBranch)
		fmt.Fprintf(os.Stderr, "remember to make a release tag asap!\n")
		releaseTag = config.BaseBranch
	}

	err = git.GitRun(gitDir, "branch", args[0], releaseTag)
	if err != nil {
		return err
	}

	fmt.Printf("created topic branch %s based on %s\n", args[0], releaseTag)
	return nil
}
