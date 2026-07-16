package app

import (
	"log/slog"
	"testing"

	"github.com/chenyme/grok2api/backend/internal/infra/config"
)

func TestMaxBatchConcurrencyUsesConfiguredValues(t *testing.T) {
	got := maxBatchConcurrency(config.BatchConfig{
		ImportConcurrency: 3, ConversionConcurrency: 7,
		SyncConcurrency: 5, RefreshConcurrency: 12,
	})
	if got != 12 {
		t.Fatalf("max = %d want 12", got)
	}
}

func TestWarnBatchVsPostgresDoesNotPanic(t *testing.T) {
	// Smoke: high settings only warn; concurrency stays as configured by caller.
	cfg := config.Config{
		Database: config.DatabaseConfig{
			Driver:   "postgres",
			Postgres: config.PostgresDatabaseConfig{MaxOpenConns: 20},
		},
		Batch: config.BatchConfig{
			ImportConcurrency: 25, ConversionConcurrency: 25,
			SyncConcurrency: 25, RefreshConcurrency: 25,
		},
	}
	warnBatchVsPostgres(slog.Default(), cfg)
	if cfg.Batch.RefreshConcurrency != 25 {
		t.Fatalf("settings must not be mutated, got refresh=%d", cfg.Batch.RefreshConcurrency)
	}
}
