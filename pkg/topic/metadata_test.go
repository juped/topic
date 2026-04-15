package topic_test

import (
	"reflect"
	"testing"
	"time"

	"go.topic.tools/topic/internal/testutil"
	"go.topic.tools/topic/pkg/topic"
	"gopkg.in/yaml.v3"
)

func TestMetadataYAMLRoundtrip(t *testing.T) {
	original := &topic.MetadataFile{
		Topics: []topic.TopicMetadata{
			{
				Name:         "ray/my-feature",
				Hash:         "abc123def456",
				Author:       "test@example.com",
				Revision:     3,
				Previous:     []string{"def456abc123", "ghi789abc123"},
				Dependencies: []string{"ray/other-feature"},
			},
		},
		Reconciliations: []topic.ReconciliationMetadata{
			{
				Id:              "rec-001",
				Time:            time.Now().UTC(),
				FirstTopic:      "ray/feature-a",
				SecondTopic:     "ray/feature-b",
				FirstTopicHash:  "123123abcabc",
				SecondTopicHash: "456456defdef",
				Author:          "test@example.com",
				Hash:            "xyz000abc123",
				AutoHash:        "000111222333",
			},
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got topic.MetadataFile
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !reflect.DeepEqual(*original, got) {
		t.Errorf("roundtrip mismatch:\nwant: %+v\n got: %+v", *original, got)
	}
}

func TestLoadWriteMetadataRoundtrip(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	testutil.Run(t, dir, "git", "branch", "test-metadata")

	cfg := &topic.Config{SyncBranch: "test-metadata"}

	want := &topic.MetadataFile{
		Topics: []topic.TopicMetadata{
			{
				Name:         "ray/my-feature",
				Hash:         "abc123def456abc123def456abc123def456abc1",
				Author:       "test@example.com",
				Revision:     2,
				Previous:     []string{"000000def456abc123def456abc123def456abc1"},
				Dependencies: []string{"ray/other-feature"},
			},
		},
		Reconciliations: []topic.ReconciliationMetadata{
			{
				Id:              "rec-001",
				Time:            time.Now().UTC(),
				FirstTopic:      "ray/feature-a",
				SecondTopic:     "ray/feature-b",
				FirstTopicHash:  "123123123456456456abcabcabcdefdefdef1234",
				SecondTopicHash: "abcabcabcdefdefdef123123123456456456abcd",
				Author:          "test@example.com",
				Hash:            "ffffdef456abc123def456abc123def456abc123",
				AutoHash:        "abc123def456abc123def456abc123def456abc1",
			},
		},
	}

	if err := topic.WriteMetadata(cfg, want); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	got, err := topic.LoadMetadata(cfg)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("roundtrip mismatch:\nwant: %+v\n got: %+v", want, got)
	}
}

func TestLoadMetadata_Empty(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	testutil.Run(t, dir, "git", "branch", "empty-metadata")

	cfg := &topic.Config{SyncBranch: "empty-metadata"}

	got, err := topic.LoadMetadata(cfg)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	if len(got.Topics) != 0 {
		t.Errorf("Topics: got %d, want 0", len(got.Topics))
	}
	if len(got.Reconciliations) != 0 {
		t.Errorf("Reconciliations: got %d, want 0", len(got.Reconciliations))
	}
}
