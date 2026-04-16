package topic

import (
	"errors"
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
		config.MetadataBranch+":topic-metadata.yaml")
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
		config.MetadataBranch)
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
		err = git.GitRun(gitDir, "update-ref", "refs/heads/"+config.MetadataBranch,
			commitHash, prevMetadataCommit)
	} else {
		err = git.GitRun(gitDir, "update-ref", "refs/heads/"+config.MetadataBranch,
			commitHash)
	}
	if err != nil {
		return err
	}

	return nil
}

func AppendTopic(config *Config, name string, hash string,
	author string, dependencies []string) error {
	topicMetadata := TopicMetadata{
		Name:         name,
		Hash:         hash,
		Author:       author,
		Revision:     1,
		Previous:     []string{},
		Dependencies: dependencies,
	}

	metadata, err := LoadMetadata(config)
	if err != nil {
		return err
	}

	metadata.Topics = append(metadata.Topics, topicMetadata)
	return WriteMetadata(config, metadata)
}

// todo: author updates maybe?
// todo: dependencies should be able to change
func UpdateTopic(config *Config, name string, hash string) error {
	metadata, err := LoadMetadata(config)
	if err != nil {
		return err
	}

	topicMetadata := GetTopicByName(metadata, name)
	if topicMetadata == nil {
		// todo: fall back on append?
		return errors.New("can't update a topic that doesn't exist")
	}

	topicMetadata.Previous = append(topicMetadata.Previous, topicMetadata.Hash)
	topicMetadata.Hash = hash
	topicMetadata.Revision += 1

	return WriteMetadata(config, metadata)
}

func AppendReconciliation(config *Config,
	reconciliationMetadata ReconciliationMetadata) error {
	metadata, err := LoadMetadata(config)
	if err != nil {
		return err
	}

	metadata.Reconciliations = append(metadata.Reconciliations,
		reconciliationMetadata)

	return WriteMetadata(config, metadata)
}

func GetTopicByName(metadata *MetadataFile, name string) *TopicMetadata {
	for i := range metadata.Topics {
		if metadata.Topics[i].Name == name {
			return &metadata.Topics[i]
		}
	}
	return nil
}

func GetTopicByHash(metadata *MetadataFile, hash string) *TopicMetadata {
	for i := range metadata.Topics {
		if metadata.Topics[i].Hash == hash {
			// todo: we expect exactly one, figure out if that's right
			return &metadata.Topics[i]
		}
	}
	return nil
}

func GetReconciliationForHashPair(metadata *MetadataFile, hash1 string,
	hash2 string) *ReconciliationMetadata {
	for _, r := range metadata.Reconciliations {
		if r.FirstTopicHash == hash1 {
			if r.SecondTopicHash == hash2 {
				return &r
			}
		} else if r.SecondTopicHash == hash1 {
			if r.FirstTopicHash == hash2 {
				return &r
			}
		}
	}
	return nil
}

func GetLatestReconciliation(metadata *MetadataFile, topic1 string,
	topic2 string) *ReconciliationMetadata {
	recs := []*ReconciliationMetadata{}

	for _, r := range metadata.Reconciliations {
		if r.FirstTopic == topic1 {
			if r.SecondTopic == topic2 {
				recs = append(recs, &r)
			}
		} else if r.SecondTopic == topic1 {
			if r.FirstTopic == topic2 {
				recs = append(recs, &r)
			}
		}
	}

	if len(recs) > 0 {
		return recs[len(recs)-1]
	}
	return nil
}
