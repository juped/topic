package topic

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"go.topic.tools/topic/pkg/git"

	"github.com/AlecAivazis/survey/v2"
	"gopkg.in/ini.v1"
)

type Config struct {
	// The branch from which releases are made, used to find base points.
	// If not set, tries in order:
	// - last component of [SyncRemote]/HEAD if Sync is true
	//   (very likely to be correct based on real-world patterns, but not 100%)
	// - base (our preferred name)
	// - main (if someone has both "main" and "master", they probably meant
	//         for "main" to supersede "master")
	// - master
	// "release" might be one to try, but it's not as conventional.
	// things like "develop" indicate totally incompatible workflows are
	// likely in use; fine if there's also a release branch, but we're not
	// going to try to autodetect them as bases.
	BaseBranch string
	// Whether to sync to a remote. Defaults to true if SyncRemote exists.
	Sync bool
	// The metadata branch to push our metadata to on the remote.
	// Defaults to "_topic-metadata" if not set.
	SyncBranch string
	// The remote to sync to. Defaults to "origin" if not set.
	SyncRemote string
	// Whether to trace git commands. Defaults to false if not set.
	Trace bool
}

func LoadConfig() (*Config, error) {
	config := &Config{}

	// common because we store our own configuration in one place,
	// not per worktree.
	gitDir, err := git.CommonGitDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(gitDir, "topic", "config")

	cfgIni, err := ini.Load(configPath)
	if err != nil {
		return createDefaultConfig(config, gitDir, string(configPath))
	}

	section := cfgIni.Section("topic")

	config.SyncRemote = section.Key("syncRemote").MustString("origin")
	syncRemoteExists := git.RemoteExists(gitDir, config.SyncRemote)
	config.Sync = section.Key("sync").MustBool(syncRemoteExists)
	config.SyncBranch = section.Key("syncBranch").MustString("_topic-metadata")

	if section.HasKey("baseBranch") {
		config.BaseBranch = section.Key("baseBranch").MustString("")
	} else {
		if config.Sync {
			name, err := git.RemoteDefaultBranch(gitDir, config.SyncRemote)
			if err != nil {
				name, err = tryBranchNames(gitDir)
				if err != nil {
					return nil, err
				}
			}
			config.BaseBranch = name
		} else {
			name, err := tryBranchNames(gitDir)
			if err != nil {
				return nil, err
			}
			config.BaseBranch = name
		}
	}

	if section.HasKey("trace") {
		config.Trace = section.Key("trace").MustBool(false)
		git.Trace = config.Trace
	}

	return config, nil
}

func tryBranchNames(gitDir string) (string, error) {
	names := []string{"base", "main", "master"}
	for _, name := range names {
		if git.BranchExists(gitDir, name) {
			return name, nil
		}
	}

	return "", errors.New("couldn't detect a base branch")
}

func createDefaultConfig(config *Config, gitDir string, configPath string) (*Config, error) {
	println("No configuration found, asking first-time setup questions.")

	prompt := &survey.Confirm{
		Message: "Should this tool sync to a remote?",
	}
	survey.AskOne(prompt, &config.Sync)

	if config.Sync {
		prompt := &survey.Input{
			Message: "Which remote should this tool use?",
			Default: "origin",
		}
		survey.AskOne(prompt, &config.SyncRemote)

		prompt = &survey.Input{
			Message: "Which branch should this tool use to sync metadata?",
			Default: "_topic-metadata",
		}
		survey.AskOne(prompt, &config.SyncBranch)

		baseBranch, err := git.RemoteDefaultBranch(gitDir, config.SyncRemote)
		if err != nil {
			baseBranch, _ = tryBranchNames(gitDir)
		}
		prompt = &survey.Input{
			Message: "Which branch do you make releases from?",
			Default: baseBranch,
		}
		survey.AskOne(prompt, &config.BaseBranch)
	} else {
		baseBranch, _ := tryBranchNames(gitDir)
		prompt := &survey.Input{
			Message: "Which branch do you make releases from?",
			Default: baseBranch,
		}
		survey.AskOne(prompt, &config.BaseBranch)
	}

	err := saveConfig(config, configPath)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func saveConfig(config *Config, configPath string) error {
	dir := filepath.Dir(configPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	iniCfg := ini.Empty()
	section := iniCfg.Section("topic")

	section.Key("sync").SetValue(strconv.FormatBool(config.Sync))
	section.Key("baseBranch").SetValue(config.BaseBranch)
	if config.SyncRemote != "" {
		section.Key("syncRemote").SetValue(config.SyncRemote)
	}
	if config.SyncBranch != "" {
		section.Key("syncBranch").SetValue(config.SyncBranch)
	}

	return iniCfg.SaveTo(configPath)
}
