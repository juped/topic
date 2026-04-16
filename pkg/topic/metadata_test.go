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

func TestAppendTopic(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	testutil.Run(t, dir, "git", "branch", "test-append-topic")

	cfg := &topic.Config{SyncBranch: "test-append-topic"}

	err := topic.AppendTopic(cfg, "ray/feature", "abc123def456abc123def456abc123def456abc1",
		"test@example.com", []string{"ray/dep"})
	if err != nil {
		t.Fatalf("AppendTopic: %v", err)
	}

	got, err := topic.LoadMetadata(cfg)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	if len(got.Topics) != 1 {
		t.Fatalf("Topics: got %d, want 1", len(got.Topics))
	}
	tm := got.Topics[0]
	if tm.Name != "ray/feature" {
		t.Errorf("Name: got %q, want %q", tm.Name, "ray/feature")
	}
	if tm.Hash != "abc123def456abc123def456abc123def456abc1" {
		t.Errorf("Hash: got %q", tm.Hash)
	}
	if tm.Author != "test@example.com" {
		t.Errorf("Author: got %q", tm.Author)
	}
	if tm.Revision != 1 {
		t.Errorf("Revision: got %d, want 1", tm.Revision)
	}
	if len(tm.Previous) != 0 {
		t.Errorf("Previous: got %v, want empty", tm.Previous)
	}
	if !reflect.DeepEqual(tm.Dependencies, []string{"ray/dep"}) {
		t.Errorf("Dependencies: got %v, want [ray/dep]", tm.Dependencies)
	}
}

func TestAppendTopic_PreservesExisting(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	testutil.Run(t, dir, "git", "branch", "test-append-preserve")

	cfg := &topic.Config{SyncBranch: "test-append-preserve"}

	if err := topic.AppendTopic(cfg, "ray/first", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1",
		"a@example.com", nil); err != nil {
		t.Fatalf("AppendTopic first: %v", err)
	}
	if err := topic.AppendTopic(cfg, "ray/second", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb2",
		"b@example.com", nil); err != nil {
		t.Fatalf("AppendTopic second: %v", err)
	}

	got, err := topic.LoadMetadata(cfg)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	if len(got.Topics) != 2 {
		t.Fatalf("Topics: got %d, want 2", len(got.Topics))
	}
	if got.Topics[0].Name != "ray/first" {
		t.Errorf("Topics[0].Name: got %q, want %q", got.Topics[0].Name, "ray/first")
	}
	if got.Topics[1].Name != "ray/second" {
		t.Errorf("Topics[1].Name: got %q, want %q", got.Topics[1].Name, "ray/second")
	}
}

func TestUpdateTopic(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	testutil.Run(t, dir, "git", "branch", "test-update-topic")

	cfg := &topic.Config{SyncBranch: "test-update-topic"}

	oldHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"
	newHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb2"

	if err := topic.AppendTopic(cfg, "ray/feature", oldHash, "test@example.com", nil); err != nil {
		t.Fatalf("AppendTopic: %v", err)
	}

	if err := topic.UpdateTopic(cfg, "ray/feature", newHash); err != nil {
		t.Fatalf("UpdateTopic: %v", err)
	}

	got, err := topic.LoadMetadata(cfg)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	if len(got.Topics) != 1 {
		t.Fatalf("Topics: got %d, want 1", len(got.Topics))
	}
	tm := got.Topics[0]
	if tm.Hash != newHash {
		t.Errorf("Hash: got %q, want %q", tm.Hash, newHash)
	}
	if tm.Revision != 2 {
		t.Errorf("Revision: got %d, want 2", tm.Revision)
	}
	if !reflect.DeepEqual(tm.Previous, []string{oldHash}) {
		t.Errorf("Previous: got %v, want [%s]", tm.Previous, oldHash)
	}
}

func TestUpdateTopic_NotFound(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	testutil.Run(t, dir, "git", "branch", "test-update-notfound")

	cfg := &topic.Config{SyncBranch: "test-update-notfound"}

	err := topic.UpdateTopic(cfg, "ray/nonexistent", "abc123def456abc123def456abc123def456abc1")
	if err == nil {
		t.Error("expected error for non-existent topic, got nil")
	}
}

func TestAppendReconciliation(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	testutil.Run(t, dir, "git", "branch", "test-append-rec")

	cfg := &topic.Config{SyncBranch: "test-append-rec"}

	rec := topic.ReconciliationMetadata{
		Id:              "rec-001",
		Time:            time.Now().UTC(),
		FirstTopic:      "ray/feature-a",
		SecondTopic:     "ray/feature-b",
		FirstTopicHash:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SecondTopicHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Author:          "test@example.com",
		Hash:            "cccccccccccccccccccccccccccccccccccccccc",
		AutoHash:        "dddddddddddddddddddddddddddddddddddddddd",
	}

	if err := topic.AppendReconciliation(cfg, rec); err != nil {
		t.Fatalf("AppendReconciliation: %v", err)
	}

	got, err := topic.LoadMetadata(cfg)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	if len(got.Reconciliations) != 1 {
		t.Fatalf("Reconciliations: got %d, want 1", len(got.Reconciliations))
	}
	if !reflect.DeepEqual(got.Reconciliations[0], rec) {
		t.Errorf("Reconciliation mismatch:\nwant: %+v\n got: %+v", rec, got.Reconciliations[0])
	}
}

func TestGetTopicByName(t *testing.T) {
	metadata := &topic.MetadataFile{
		Topics: []topic.TopicMetadata{
			{Name: "ray/alpha", Hash: "aaaa"},
			{Name: "ray/beta", Hash: "bbbb"},
		},
	}

	got := topic.GetTopicByName(metadata, "ray/beta")
	if got == nil {
		t.Fatal("expected topic, got nil")
	}
	if got.Name != "ray/beta" {
		t.Errorf("Name: got %q, want %q", got.Name, "ray/beta")
	}

	if topic.GetTopicByName(metadata, "ray/missing") != nil {
		t.Error("expected nil for missing topic")
	}
}

func TestGetTopicByName_PointerIsSliceElement(t *testing.T) {
	metadata := &topic.MetadataFile{
		Topics: []topic.TopicMetadata{
			{Name: "ray/alpha", Hash: "aaaa"},
		},
	}

	got := topic.GetTopicByName(metadata, "ray/alpha")
	if got == nil {
		t.Fatal("expected topic, got nil")
	}
	got.Hash = "zzzz"

	if metadata.Topics[0].Hash != "zzzz" {
		t.Errorf("modification through pointer not reflected in slice: got %q, want %q",
			metadata.Topics[0].Hash, "zzzz")
	}
}

func TestGetTopicByHash(t *testing.T) {
	metadata := &topic.MetadataFile{
		Topics: []topic.TopicMetadata{
			{Name: "ray/alpha", Hash: "aaaa"},
			{Name: "ray/beta", Hash: "bbbb"},
		},
	}

	got := topic.GetTopicByHash(metadata, "aaaa")
	if got == nil {
		t.Fatal("expected topic, got nil")
	}
	if got.Name != "ray/alpha" {
		t.Errorf("Name: got %q, want %q", got.Name, "ray/alpha")
	}

	if topic.GetTopicByHash(metadata, "xxxx") != nil {
		t.Error("expected nil for missing hash")
	}
}

func TestGetReconciliationForHashPair(t *testing.T) {
	metadata := &topic.MetadataFile{
		Reconciliations: []topic.ReconciliationMetadata{
			{FirstTopicHash: "aaaa", SecondTopicHash: "bbbb", Id: "rec-1"},
		},
	}

	// Normal order.
	got := topic.GetReconciliationForHashPair(metadata, "aaaa", "bbbb")
	if got == nil {
		t.Fatal("expected reconciliation, got nil")
	}
	if got.Id != "rec-1" {
		t.Errorf("Id: got %q, want %q", got.Id, "rec-1")
	}

	// Reversed order.
	got = topic.GetReconciliationForHashPair(metadata, "bbbb", "aaaa")
	if got == nil {
		t.Fatal("expected reconciliation for reversed pair, got nil")
	}
	if got.Id != "rec-1" {
		t.Errorf("Id (reversed): got %q, want %q", got.Id, "rec-1")
	}

	// Missing.
	if topic.GetReconciliationForHashPair(metadata, "aaaa", "xxxx") != nil {
		t.Error("expected nil for missing hash pair")
	}
}

func TestGetLatestReconciliation(t *testing.T) {
	metadata := &topic.MetadataFile{
		Reconciliations: []topic.ReconciliationMetadata{
			{FirstTopic: "ray/a", SecondTopic: "ray/b", Id: "rec-1"},
			{FirstTopic: "ray/a", SecondTopic: "ray/b", Id: "rec-2"},
			{FirstTopic: "ray/a", SecondTopic: "ray/b", Id: "rec-3"},
		},
	}

	got := topic.GetLatestReconciliation(metadata, "ray/a", "ray/b")
	if got == nil {
		t.Fatal("expected reconciliation, got nil")
	}
	if got.Id != "rec-3" {
		t.Errorf("Id: got %q, want %q (should return latest)", got.Id, "rec-3")
	}
}

func TestGetLatestReconciliation_ReversedTopicOrder(t *testing.T) {
	metadata := &topic.MetadataFile{
		Reconciliations: []topic.ReconciliationMetadata{
			{FirstTopic: "ray/a", SecondTopic: "ray/b", Id: "rec-1"},
			{FirstTopic: "ray/b", SecondTopic: "ray/a", Id: "rec-2"},
		},
	}

	got := topic.GetLatestReconciliation(metadata, "ray/a", "ray/b")
	if got == nil {
		t.Fatal("expected reconciliation, got nil")
	}
	if got.Id != "rec-2" {
		t.Errorf("Id: got %q, want %q", got.Id, "rec-2")
	}
}

func TestGetLatestReconciliation_Missing(t *testing.T) {
	metadata := &topic.MetadataFile{
		Reconciliations: []topic.ReconciliationMetadata{
			{FirstTopic: "ray/a", SecondTopic: "ray/b", Id: "rec-1"},
		},
	}

	if topic.GetLatestReconciliation(metadata, "ray/a", "ray/c") != nil {
		t.Error("expected nil for missing topic pair")
	}
}
