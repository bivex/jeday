package config

import (
	"testing"
	"time"
)

func TestLoadDefaultsForTuningConfig(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ServerPrefork {
		t.Fatalf("ServerPrefork = true, want false by default")
	}
	if cfg.RegistrationBatchSize != 100 {
		t.Fatalf("RegistrationBatchSize = %d, want 100", cfg.RegistrationBatchSize)
	}
	if cfg.RegistrationBatchWait != 10*time.Millisecond {
		t.Fatalf("RegistrationBatchWait = %s, want 10ms", cfg.RegistrationBatchWait)
	}
	if cfg.WorkerInterval != 2*time.Second {
		t.Fatalf("WorkerInterval = %s, want 2s", cfg.WorkerInterval)
	}
	if cfg.WorkerUpgradeLimit != 10 {
		t.Fatalf("WorkerUpgradeLimit = %d, want 10", cfg.WorkerUpgradeLimit)
	}
}

func TestLoadParsesTuningConfigFromEnv(t *testing.T) {
	t.Setenv("SERVER_PREFORK", "true")
	t.Setenv("REGISTRATION_BATCH_SIZE", "250")
	t.Setenv("REGISTRATION_BATCH_WAIT", "5ms")
	t.Setenv("WORKER_INTERVAL", "7s")
	t.Setenv("WORKER_UPGRADE_LIMIT", "3")

	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.ServerPrefork {
		t.Fatalf("ServerPrefork = false, want true")
	}
	if cfg.RegistrationBatchSize != 250 {
		t.Fatalf("RegistrationBatchSize = %d, want 250", cfg.RegistrationBatchSize)
	}
	if cfg.RegistrationBatchWait != 5*time.Millisecond {
		t.Fatalf("RegistrationBatchWait = %s, want 5ms", cfg.RegistrationBatchWait)
	}
	if cfg.WorkerInterval != 7*time.Second {
		t.Fatalf("WorkerInterval = %s, want 7s", cfg.WorkerInterval)
	}
	if cfg.WorkerUpgradeLimit != 3 {
		t.Fatalf("WorkerUpgradeLimit = %d, want 3", cfg.WorkerUpgradeLimit)
	}
}
