package cmd

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "topic",
	Short: "Manage git topics",
	Long: `Manage git topics.

This tool primarily orchestrates git commands; you can see what it's invoking
by setting the TOPIC_TRACE environment variable. (But there are usually more
compact commands available to you, as a user rather than an automated tool.)`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		os.Exit(0)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of this tool",
	Long:  `Print the version of this tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		if info, ok := debug.ReadBuildInfo(); ok {
			fmt.Println(info.Main.Version)
		} else {
			fmt.Println("unknown")
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
