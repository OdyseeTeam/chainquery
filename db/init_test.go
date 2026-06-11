package db

import (
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestPrepareDSNPreservesExistingParams(t *testing.T) {
	ConfigureConnection(12, 4, time.Minute, 3*time.Second, 4*time.Second, 5*time.Second)

	dsn, err := prepareDSN("user:pass@tcp(localhost:3306)/chainquery?timeout=7s&readTimeout=8s&loc=UTC&multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timeout != 7*time.Second {
		t.Fatalf("expected existing timeout to be preserved, got %s", cfg.Timeout)
	}
	if cfg.ReadTimeout != 8*time.Second {
		t.Fatalf("expected existing read timeout to be preserved, got %s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 5*time.Second {
		t.Fatalf("expected default write timeout to be applied, got %s", cfg.WriteTimeout)
	}
	if !cfg.ParseTime {
		t.Fatal("expected parseTime to be enabled")
	}
	if !cfg.MultiStatements {
		t.Fatal("expected existing multiStatements param to be preserved")
	}
	if strings.Contains(dsn, "??") {
		t.Fatalf("dsn contains malformed query separator: %s", dsn)
	}
}
