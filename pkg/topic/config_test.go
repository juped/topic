package topic_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.topic.tools/topic/internal/testutil"
	"go.topic.tools/topic/pkg/topic"
)

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	configDir := filepath.Join(dir, ".git", "topic")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestLoadConfig_ExplicitFields(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	writeConfig(t, dir, `[topic]
baseBranch = base
sync = true
syncRemote = upstream
metadataBranch = _meta
`)

	cfg, err := topic.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.BaseBranch != "base" {
		t.Errorf("BaseBranch: got %q, want %q", cfg.BaseBranch, "base")
	}
	if !cfg.Sync {
		t.Error("Sync: got false, want true")
	}
	if cfg.SyncRemote != "upstream" {
		t.Errorf("SyncRemote: got %q, want %q", cfg.SyncRemote, "upstream")
	}
	if cfg.MetadataBranch != "_meta" {
		t.Errorf("MetadataBranch: got %q, want %q", cfg.MetadataBranch, "_meta")
	}
}

func TestLoadConfig_SyncDefaults(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	// sync=true but no syncRemote/metadataBranch; should fall back to defaults.
	writeConfig(t, dir, `[topic]
baseBranch = base
sync = true
`)

	cfg, err := topic.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.SyncRemote != "origin" {
		t.Errorf("SyncRemote default: got %q, want origin", cfg.SyncRemote)
	}
	if cfg.MetadataBranch != "_topic-metadata" {
		t.Errorf("MetadataBranch default: got %q, want _topic-metadata",
			cfg.MetadataBranch)
	}
}

func TestLoadConfig_SyncFalse(t *testing.T) {
	dir := testutil.SetupRepo(t)
	t.Chdir(dir)
	writeConfig(t, dir, `[topic]
baseBranch = base
sync = false
`)

	cfg, err := topic.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Sync {
		t.Error("Sync: got true, want false")
	}
	// SyncRemote/MetadataBranch should be empty when sync=false.
	if cfg.SyncRemote != "" {
		t.Errorf("SyncRemote: got %q, want empty (sync disabled)", cfg.SyncRemote)
	}
}
