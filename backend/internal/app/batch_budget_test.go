package app

import (
	"testing"

	"github.com/chenyme/grok2api/backend/internal/infra/config"
)

func TestClampBatchForDatabaseReservesRequestPath(t *testing.T) {
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
	got := clampBatchForDatabase(cfg)
	// half of 20 = 10 budget; refresh capped further.
	if got.ImportConcurrency != 10 || got.ConversionConcurrency != 10 || got.SyncConcurrency != 10 {
		t.Fatalf("import/sync/conversion = %#v", got)
	}
	if got.RefreshConcurrency != 8 {
		t.Fatalf("refresh = %d want 8", got.RefreshConcurrency)
	}

	// SQLite must not clamp.
	cfg.Database.Driver = "sqlite"
	got = clampBatchForDatabase(cfg)
	if got.RefreshConcurrency != 25 {
		t.Fatalf("sqlite refresh clamped unexpectedly: %d", got.RefreshConcurrency)
	}
}
