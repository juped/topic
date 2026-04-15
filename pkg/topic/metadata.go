package topic

import (
	"time"

	"go.topic.tools/topic/pkg/git"
	"gopkg.in/yaml.v3"
)

type TopicMetadata struct {
	// The topic's branch name.
	Name string
	// Hash at the tip.
	Hash string
	// Author of the topic.
	Author string
	// Iteration we're on.
	Revision int
	// History of previous tips.
	Previous []string
	// Other topics this topic depends on.
	Dependencies []string
}

type ReconciliationMetadata struct {
	// Unique ID of this reconciliation.
	Id string
	// UTC timestamp of this reconciliation.
	Time time.Time
	// Topics reconciled.
	FirstTopic  string
	SecondTopic string
	// Tip hashes as of the reconciliation.
	FirstTopicHash  string
	SecondTopicHash string
	// Author of the reconciliation.
	Author string
	// Hash of the reconciliation commit.
	Hash string
	// Hash of the automerge we started reconciliation with.
	AutoHash string
}

type MetadataFile struct {
	// Topics in the metadata file.
	Topics []TopicMetadata
	// Reconciliations in the metadata file.
	Reconciliations []ReconciliationMetadata
}

func LoadMetadata(config *Config) (*MetadataFile, error) {
	metadata := &MetadataFile{
		Topics:          []TopicMetadata{},
		Reconciliations: []ReconciliationMetadata{},
	}

	gitDir, err := git.GitDir()
	if err != nil {
		return nil, err
	}

	metadataYaml, err := git.GitCommand(gitDir, "cat-file", "blob",
		config.SyncBranch+":topic-metadata.yaml")
	if err != nil {
		// Doesn't exist yet. Use empty metadata.
		// todo: handle other error cases
		return metadata, nil
	}

	err = yaml.Unmarshal([]byte(metadataYaml), metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// todo: merges of the metadata file (topics merge, reconciliations append-only)
func WriteMetadata(config *Config, metadata *MetadataFile) error {
	gitDir, err := git.GitDir()
	if err != nil {
		return err
	}

	metadataYaml, err := yaml.Marshal(metadata)

	blobHash, err := git.GitPipe(gitDir, string(metadataYaml), "hash-object", "-w",
		"--stdin")
	if err != nil {
		return err
	}

	// todo: consider -z format
	treeEntry := "100644 blob " + blobHash + "\ttopic-metadata.yaml\n"
	treeHash, err := git.GitPipe(gitDir, treeEntry, "mktree")
	if err != nil {
		return err
	}

	prevMetadataCommit, err := git.GitCommand(gitDir, "rev-parse",
		config.SyncBranch)
	branchExists := err == nil

	var commitHash string
	if branchExists {
		// todo: extra parents for reconciliation reachability
		commitHash, err = git.GitCommand(gitDir, "commit-tree", treeHash, "-p",
			prevMetadataCommit)
	} else {
		commitHash, err = git.GitCommand(gitDir, "commit-tree", treeHash)
	}
	if err != nil {
		return err
	}

	if branchExists {
		err = git.GitRun(gitDir, "update-ref", "refs/heads/"+config.SyncBranch,
			commitHash, prevMetadataCommit)
	} else {
		err = git.GitRun(gitDir, "update-ref", "refs/heads/"+config.SyncBranch,
			commitHash)
	}
	if err != nil {
		return err
	}

	return nil
}
